// handlers/finance.go
package handlers

import (
	"finance-tracker/models"
	"finance-tracker/storage"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type FinanceHandler struct {
	financeStore *storage.FinanceStorage
	workLogStore *storage.WorkLogStorage // Добавляем хранилище табеля
}

func NewFinanceHandler(financeStore *storage.FinanceStorage, workLogStore *storage.WorkLogStorage) *FinanceHandler {
	return &FinanceHandler{
		financeStore: financeStore,
		workLogStore: workLogStore,
	}
}

func (h *FinanceHandler) Index(c *gin.Context) {
	fmt.Println("Обработка запроса на главную страницу...")

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

	data := h.financeStore.GetData()
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

	// Вычисление статистики за месяц
	monthlyIncome := 0.0
	monthlyExpense := 0.0
	now := time.Now()
	oneMonthAgo := now.AddDate(0, -1, 0)

	for _, t := range data.Transactions {
		if t.DateTime.After(oneMonthAgo) {
			if t.IsPositive {
				monthlyIncome += t.Amount
			} else {
				monthlyExpense += t.Amount
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

	// Форматируем баланс
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

	// Получаем записи о работе
	workData := h.workLogStore.GetData()

	c.HTML(http.StatusOK, "index.html", gin.H{
		"balances":       balances,
		"transactions":   formattedTrans,
		"monthlyIncome":  fmt.Sprintf("%.2f", monthlyIncome),
		"monthlyExpense": fmt.Sprintf("%.2f", monthlyExpense),
		"pagination":     pagination,
		"today":          time.Now().Format("2006-01-02"),
		"workEntries":    workData.Entries, // Передаём записи о работе
	})
}

func (h *FinanceHandler) GetTransactions(c *gin.Context) {
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

	data := h.financeStore.GetData()
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

func (h *FinanceHandler) AddTransaction(c *gin.Context) {
	amountStr := c.PostForm("amount")
	description := c.PostForm("description")
	currency := c.PostForm("currency")
	notes := c.PostForm("notes")
	action := c.PostForm("action")

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Неверная сумма")
		return
	}

	if description == "" {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Описание не может быть пустым")
		return
	}

	if currency == "" {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Выберите валюту")
		return
	}

	data := h.financeStore.GetData()
	newID := 1
	if len(data.Transactions) > 0 {
		newID = data.Transactions[len(data.Transactions)-1].ID + 1
	}

	newTransaction := models.Transaction{
		ID:          newID,
		Amount:      amount,
		Description: description,
		DateTime:    time.Now(),
		IsPositive:  action == "add-income",
		Currency:    currency,
		Notes:       notes,
	}

	data.Transactions = append(data.Transactions, newTransaction)

	h.financeStore.RecalculateBalances()
	if err := h.financeStore.Save(); err != nil {
		c.Redirect(http.StatusFound, "/?message=Ошибка при сохранении данных")
		return
	}

	c.Redirect(http.StatusFound, "/?message=Транзакция добавлена")
}

// handlers/finance.go (продолжение)
func (h *FinanceHandler) EditTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Неверный ID транзакции")
		return
	}

	amountStr := c.PostForm("amount")
	description := c.PostForm("description")
	currency := c.PostForm("currency")
	notes := c.PostForm("notes")
	action := c.PostForm("action")

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Неверная сумма")
		return
	}

	if description == "" {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Описание не может быть пустым")
		return
	}

	if currency == "" {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Выберите валюту")
		return
	}

	data := h.financeStore.GetData()
	for i, t := range data.Transactions {
		if t.ID == id {
			data.Transactions[i].Amount = amount
			data.Transactions[i].Description = description
			data.Transactions[i].Currency = currency
			data.Transactions[i].Notes = notes
			data.Transactions[i].IsPositive = action == "add-income"
			break
		}
	}

	h.financeStore.RecalculateBalances()
	if err := h.financeStore.Save(); err != nil {
		c.Redirect(http.StatusFound, "/?message=Ошибка при сохранении данных")
		return
	}

	c.Redirect(http.StatusFound, "/?message=Транзакция обновлена")
}

func (h *FinanceHandler) DeleteTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/?message=Ошибка: Неверный ID транзакции")
		return
	}

	data := h.financeStore.GetData()
	for i, t := range data.Transactions {
		if t.ID == id {
			data.Transactions = append(data.Transactions[:i], data.Transactions[i+1:]...)
			break
		}
	}

	h.financeStore.RecalculateBalances()
	if err := h.financeStore.Save(); err != nil {
		c.Redirect(http.StatusFound, "/?message=Ошибка при сохранении данных")
		return
	}

	c.Redirect(http.StatusFound, "/?message=Транзакция удалена")
}
