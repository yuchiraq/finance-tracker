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
func Index(c *gin.Context) {
	fmt.Println("Обработка запроса на главную страницу...")

	// Фильтрация
	filterType := c.Query("filter-type")
	filterDateStart := c.Query("filter-date-start")
	filterDateEnd := c.Query("filter-date-end")

	filteredTrans := []models.Transaction{}
	for _, t := range data.Transactions {
		// Фильтр по типу
		if filterType != "" {
			if filterType == "income" && !t.IsPositive {
				continue
			}
			if filterType == "expense" && t.IsPositive {
				continue
			}
		}
		// Фильтр по дате
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

	// Форматирование транзакций для шаблона
	formattedTrans := make([]gin.H, len(filteredTrans))
	for i, t := range filteredTrans {
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

	// Логируем данные, передаваемые в шаблон
	fmt.Printf("Данные для шаблона:\n")
	fmt.Printf("  balances: %+v\n", balances)
	fmt.Printf("  transactions: %+v\n", formattedTrans)
	fmt.Printf("  monthlyIncome: %.2f\n", monthlyIncome)
	fmt.Printf("  monthlyExpense: %.2f\n", monthlyExpense)

	// Рендеринг страницы
	c.HTML(http.StatusOK, "index.html", gin.H{
		"balances":       balances,
		"transactions":   formattedTrans,
		"monthlyIncome":  fmt.Sprintf("%.2f", monthlyIncome),
		"monthlyExpense": fmt.Sprintf("%.2f", monthlyExpense),
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
