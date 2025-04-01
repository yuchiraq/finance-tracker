package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"finance-tracker/models"

	"github.com/gin-gonic/gin"
)

var data *models.FinanceData
var saveDataFunc func()

// Init инициализирует данные и функцию сохранения
func Init(d *models.FinanceData, saveFunc func()) {
	data = d
	saveDataFunc = saveFunc
	if data.Balances == nil {
		data.Balances = make(map[string]float64)
		fmt.Println("Handlers: Balances не инициализирован, создаём пустую карту")
	}
	fmt.Printf("Handlers: Баланс при инициализации: %+v\n", data.Balances)
	fmt.Printf("Handlers: Транзакции при инициализации: %+v\n", data.Transactions)
}

// Index обрабатывает главную страницу
// handlers/handlers.go
func Index(c *gin.Context) {
	fmt.Println("Обработка запроса на главную страницу...")

	// Получаем номер страницы из query-параметра
	pageStr := c.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	const pageSize = 10

	// Фильтрация
	filterType := c.Query("filter-type")
	filterDateStart := c.Query("filter-date-start")
	filterDateEnd := c.Query("filter-date-end")

	filteredTrans := []models.Transaction{}
	for _, t := range data.Transactions {
		if filterType != "" {
			if filterType == "income" && !t.IsPositive {
				continue
			}
			if filterType == "expense" && t.IsPositive {
				continue
			}
		}
		if filterDateStart != "" {
			startDate, err := time.Parse("2006-01-02", filterDateStart)
			if err == nil && t.DateTime.Before(startDate) {
				continue
			}
		}
		if filterDateEnd != "" {
			endDate, err := time.Parse("2006-01-02", filterDateEnd)
			if err == nil && t.DateTime.After(endDate) {
				continue
			}
		}
		filteredTrans = append(filteredTrans, t)
	}

	// Сортировка по дате (новые сверху)
	sort.Slice(filteredTrans, func(i, j int) bool {
		return filteredTrans[i].DateTime.After(filteredTrans[j].DateTime)
	})

	// Пагинация
	totalTrans := len(filteredTrans)
	totalPages := (totalTrans + pageSize - 1) / pageSize
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > totalTrans {
		end = totalTrans
	}
	paginatedTrans := filteredTrans[start:end]

	// Форматирование транзакций для шаблона
	formattedTrans := make([]gin.H, len(paginatedTrans))
	for i, t := range paginatedTrans {
		formattedTrans[i] = gin.H{
			"ID":          t.ID,
			"Amount":      t.Amount,
			"Description": t.Description,
			"DateTime":    t.DateTime.Format("02.01.2006 15:04"),
			"IsPositive":  t.IsPositive,
			"Category":    t.Category,
			"Tags":        strings.Join(t.Tags, ", "),
			"Currency":    t.Currency,
			"Notes":       t.Notes,
		}
	}

	// Вычисление статистики за месяц
	monthlyIncome := 0.0
	monthlyExpense := 0.0
	now := time.Now()
	oneMonthAgo := now.AddDate(0, -1, 0)

	for _, t := range data.Transactions {
		if t.DateTime.After(oneMonthAgo) {
			amountInBYN := convertToBYN(t.Amount, t.Currency)
			if t.IsPositive {
				monthlyIncome += amountInBYN
			} else {
				monthlyExpense += amountInBYN
			}
		}
	}

	// Сортируем валюты
	currencies := []string{}
	for currency := range data.Balances {
		if data.Balances[currency] != 0 {
			currencies = append(currencies, currency)
		}
	}
	sort.Strings(currencies)

	// Форматируем баланс для каждой валюты
	balances := []gin.H{}
	for _, currency := range currencies {
		balanceValue := data.Balances[currency]
		balances = append(balances, gin.H{
			"Currency": currency,
			"Balance":  fmt.Sprintf("%.2f", balanceValue),
		})
	}

	// Данные для пагинации
	pagination := gin.H{
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"HasPrev":     page > 1,
		"HasNext":     page < totalPages,
		"PrevPage":    page - 1,
		"NextPage":    page + 1,
	}

	// Рендеринг страницы
	c.HTML(http.StatusOK, "index.html", gin.H{
		"balances":       balances,
		"transactions":   formattedTrans,
		"monthlyIncome":  fmt.Sprintf("%.2f", monthlyIncome),
		"monthlyExpense": fmt.Sprintf("%.2f", monthlyExpense),
		"pagination":     pagination,
		"today":          time.Now().Format("2006-01-02"),
		"workEntries":    workLogData.Entries,
	})
}

// AddTransaction добавляет новую транзакцию
func AddTransaction(c *gin.Context) {
	amountStr := c.PostForm("amount")
	description := c.PostForm("description")
	action := c.PostForm("action")
	tags := c.PostForm("tags")
	currency := c.PostForm("currency")
	notes := c.PostForm("notes")

	// Определяем тип операции на основе нажатой кнопки
	var transType string
	switch action {
	case "add-income":
		transType = "income"
	case "add-expense":
		transType = "expense"
	default:
		c.Redirect(http.StatusFound, "/?message=Ошибка: Неверный тип транзакции")
		return
	}

	// Валидация
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Неправильный формат суммы")
		return
	}
	if amount < 0 {
		amount = -amount // Убедимся, что amount положительный
	}
	if description == "" || len(description) > 100 {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Описание должно быть от 1 до 100 символов")
		return
	}
	if currency == "" {
		currency = "BYN"
	}

	// Разделяем теги
	tagList := []string{}
	if tags != "" {
		tagList = strings.Split(tags, ",")
		for i := range tagList {
			tagList[i] = strings.TrimSpace(tagList[i])
		}
	}

	// Находим максимальный ID
	maxID := 0
	for _, t := range data.Transactions {
		if t.ID > maxID {
			maxID = t.ID
		}
	}

	// Создаём новую транзакцию
	isPositive := transType == "income"
	newTransaction := models.Transaction{
		ID:          maxID + 1,
		Amount:      amount,
		Description: description,
		DateTime:    time.Now(),
		IsPositive:  isPositive,
		Category:    "", // Категория больше не используется
		Tags:        tagList,
		Currency:    currency,
		Notes:       notes,
	}

	data.Transactions = append(data.Transactions, newTransaction)

	// Обновляем баланс для соответствующей валюты
	if isPositive {
		data.Balances[currency] += amount
	} else {
		data.Balances[currency] -= amount
	}

	// Очищаем нулевые балансы
	for currency, balance := range data.Balances {
		if balance == 0 {
			delete(data.Balances, currency)
		}
	}

	saveDataFunc()

	c.Redirect(http.StatusFound, "/?message=Транзакция добавлена")
}

func GetTransactions(c *gin.Context) {
	pageStr := c.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	const pageSize = 10

	// Фильтрация
	filterType := c.Query("filter-type")
	filterDateStart := c.Query("filter-date-start")
	filterDateEnd := c.Query("filter-date-end")

	filteredTrans := []models.Transaction{}
	for _, t := range data.Transactions {
		if filterType != "" {
			if filterType == "income" && !t.IsPositive {
				continue
			}
			if filterType == "expense" && t.IsPositive {
				continue
			}
		}
		if filterDateStart != "" {
			startDate, err := time.Parse("2006-01-02", filterDateStart)
			if err == nil && t.DateTime.Before(startDate) {
				continue
			}
		}
		if filterDateEnd != "" {
			endDate, err := time.Parse("2006-01-02", filterDateEnd)
			if err == nil && t.DateTime.After(endDate) {
				continue
			}
		}
		filteredTrans = append(filteredTrans, t)
	}

	// Сортировка по дате (новые сверху)
	sort.Slice(filteredTrans, func(i, j int) bool {
		return filteredTrans[i].DateTime.After(filteredTrans[j].DateTime)
	})

	// Пагинация
	totalTrans := len(filteredTrans)
	totalPages := (totalTrans + pageSize - 1) / pageSize
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > totalTrans {
		end = totalTrans
	}
	paginatedTrans := filteredTrans[start:end]

	// Форматирование транзакций
	formattedTrans := make([]gin.H, len(paginatedTrans))
	for i, t := range paginatedTrans {
		formattedTrans[i] = gin.H{
			"ID":          t.ID,
			"Amount":      fmt.Sprintf("%.2f", t.Amount),
			"Description": t.Description,
			"DateTime":    t.DateTime.Format("02.01.2006 15:04"),
			"IsPositive":  t.IsPositive,
			"Category":    t.Category,
			"Tags":        strings.Join(t.Tags, ", "),
			"Currency":    t.Currency,
			"Notes":       t.Notes,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": formattedTrans,
		"hasNext":      page < totalPages,
		"nextPage":     page + 1,
	})
}

// EditTransaction редактирует существующую транзакцию
func EditTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Неверный ID")
		return
	}

	amountStr := c.PostForm("amount")
	description := c.PostForm("description")
	action := c.PostForm("action")
	tags := c.PostForm("tags")
	currency := c.PostForm("currency")
	notes := c.PostForm("notes")

	// Определяем тип операции на основе нажатой кнопки
	var transType string
	switch action {
	case "add-income":
		transType = "income"
	case "add-expense":
		transType = "expense"
	default:
		c.Redirect(http.StatusFound, "/?message=Ошибка: Неверный тип транзакции")
		return
	}

	// Валидация
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Неправильный формат суммы")
		return
	}
	if amount < 0 {
		amount = -amount // Убедимся, что amount положительный
	}
	if description == "" || len(description) > 100 {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Описание должно быть от 1 до 100 символов")
		return
	}
	if currency == "" {
		currency = "BYN"
	}

	// Разделяем теги
	tagList := []string{}
	if tags != "" {
		tagList = strings.Split(tags, ",")
		for i := range tagList {
			tagList[i] = strings.TrimSpace(tagList[i])
		}
	}

	// Ищем транзакцию
	var foundIndex = -1
	var oldTransaction models.Transaction
	for i, t := range data.Transactions {
		if t.ID == id {
			oldTransaction = t
			foundIndex = i
			break
		}
	}

	if foundIndex == -1 {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Транзакция не найдена")
		return
	}

	// Корректируем баланс (убираем старую транзакцию)
	if oldTransaction.IsPositive {
		data.Balances[oldTransaction.Currency] -= oldTransaction.Amount
	} else {
		data.Balances[oldTransaction.Currency] += oldTransaction.Amount
	}

	// Обновляем транзакцию
	isPositive := transType == "income"
	data.Transactions[foundIndex] = models.Transaction{
		ID:          id,
		Amount:      amount,
		Description: description,
		DateTime:    oldTransaction.DateTime,
		IsPositive:  isPositive,
		Category:    "", // Категория больше не используется
		Tags:        tagList,
		Currency:    currency,
		Notes:       notes,
	}

	// Обновляем баланс для новой валюты
	if isPositive {
		data.Balances[currency] += amount
	} else {
		data.Balances[currency] -= amount
	}

	// Очищаем нулевые балансы
	for currency, balance := range data.Balances {
		if balance == 0 {
			delete(data.Balances, currency)
		}
	}

	saveDataFunc()

	c.Redirect(http.StatusFound, "/?message=Транзакция обновлена")
}

// DeleteTransaction удаляет транзакцию
func DeleteTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Неверный ID")
		return
	}

	// Ищем транзакцию
	var transactionToRemove models.Transaction
	var foundIndex = -1

	for i, t := range data.Transactions {
		if t.ID == id {
			transactionToRemove = t
			foundIndex = i
			break
		}
	}

	if foundIndex == -1 {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Транзакция не найдена")
		return
	}

	// Корректируем баланс
	if transactionToRemove.IsPositive {
		data.Balances[transactionToRemove.Currency] -= transactionToRemove.Amount
	} else {
		data.Balances[transactionToRemove.Currency] += transactionToRemove.Amount
	}

	// Очищаем нулевые балансы
	for currency, balance := range data.Balances {
		if balance == 0 {
			delete(data.Balances, currency)
		}
	}

	// Удаляем транзакцию
	data.Transactions = append(data.Transactions[:foundIndex], data.Transactions[foundIndex+1:]...)

	saveDataFunc()

	c.Redirect(http.StatusFound, "/?message=Транзакция удалена")
}

// convertToBYN конвертирует сумму в BYN (используется только для статистики)
func convertToBYN(amount float64, currency string) float64 {
	rates := map[string]float64{
		"BYN": 1.0,
		"USD": 3.3,
		"EUR": 3.5,
	}
	rate, exists := rates[currency]
	if !exists {
		rate = 1.0 // По умолчанию
	}
	return amount * rate
}

// handlers/handlers.go
var workLogData *models.WorkLogData
var saveWorkLogFunc func()

func InitWorkLog(w *models.WorkLogData, saveFunc func()) {
	workLogData = w
	saveWorkLogFunc = saveFunc
}

// handlers/handlers.go
func WorkLog(c *gin.Context) {
	// Форматируем записи с дополнительной информацией
	formattedEntries := []gin.H{}
	for _, entry := range workLogData.Entries {
		date, _ := time.Parse("2006-01-02", entry.Date)
		dayOfWeek := map[string]string{
			"Monday":    "Понедельник",
			"Tuesday":   "Вторник",
			"Wednesday": "Среда",
			"Thursday":  "Четверг",
			"Friday":    "Пятница",
			"Saturday":  "Суббота",
			"Sunday":    "Воскресенье",
		}[date.Weekday().String()]
		formattedDate := fmt.Sprintf("%s, %d %s", dayOfWeek, date.Day(), map[string]string{
			"January":   "января",
			"February":  "февраля",
			"March":     "марта",
			"April":     "апреля",
			"May":       "мая",
			"June":      "июня",
			"July":      "июля",
			"August":    "августа",
			"September": "сентября",
			"October":   "октября",
			"November":  "ноября",
			"December":  "декабря",
		}[date.Month().String()])

		var hoursWorked string
		if !entry.IsDayOff {
			start, _ := time.Parse("15:04", entry.StartTime)
			end, _ := time.Parse("15:04", entry.EndTime)
			duration := end.Sub(start).Hours()
			if duration < 0 {
				duration += 24 // Учитываем переход через полночь
			}
			hoursWorked = fmt.Sprintf("%.1f часов", duration)
		}

		formattedEntries = append(formattedEntries, gin.H{
			"Date":          entry.Date,
			"FormattedDate": formattedDate,
			"Place":         entry.Place,
			"StartTime":     entry.StartTime,
			"EndTime":       entry.EndTime,
			"IsDayOff":      entry.IsDayOff,
			"HoursWorked":   hoursWorked,
		})
	}

	// Сортируем записи по дате (новые сверху)
	sort.Slice(formattedEntries, func(i, j int) bool {
		return formattedEntries[i]["Date"].(string) > formattedEntries[j]["Date"].(string)
	})

	c.HTML(http.StatusOK, "worklog.html", gin.H{
		"entries": formattedEntries,
	})
}

func AddWork(c *gin.Context) {
	place := c.PostForm("place")
	startTime := c.PostForm("start_time")
	endTime := c.PostForm("end_time")
	isDayOff := c.PostForm("is_day_off") == "on"

	today := time.Now().Format("2006-01-02")

	// Проверяем, есть ли запись за сегодня
	for _, entry := range workLogData.Entries {
		if entry.Date == today {
			c.Redirect(http.StatusFound, "/?message=Запись за сегодня уже существует")
			return
		}
	}

	// Валидация
	if !isDayOff {
		if place == "" {
			c.Redirect(http.StatusFound, "/?message=Ошибка: Укажите место работы")
			return
		}
		if startTime == "" || endTime == "" {
			c.Redirect(http.StatusFound, "/?message=Ошибка: Укажите время работы")
			return
		}
		_, err := time.Parse("15:04", startTime)
		if err != nil {
			c.Redirect(http.StatusFound, "/?message=Ошибка: Неверный формат времени начала")
			return
		}
		_, err = time.Parse("15:04", endTime)
		if err != nil {
			c.Redirect(http.StatusFound, "/?message=Ошибка: Неверный формат времени окончания")
			return
		}
	}

	// Добавляем запись
	newEntry := models.WorkEntry{
		Date:      today,
		Place:     place,
		StartTime: startTime,
		EndTime:   endTime,
		IsDayOff:  isDayOff,
	}
	workLogData.Entries = append(workLogData.Entries, newEntry)

	saveWorkLogFunc()

	c.Redirect(http.StatusFound, "/?message=Запись о работе добавлена")
}
