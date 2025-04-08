package handlers

import (
	"encoding/json"
	"finance-tracker/models"
	"finance-tracker/storage"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	financeStore *storage.FinanceStorage
}

func NewStatsHandler(financeStore *storage.FinanceStorage) *StatsHandler {
	return &StatsHandler{financeStore: financeStore}
}

func (h *StatsHandler) Stats(c *gin.Context) {
	period := c.Query("period")
	dateStr := c.Query("date")

	if period == "" {
		period = "month"
	}
	if dateStr == "" {
		if period == "month" {
			dateStr = time.Now().Format("2006-01")
		} else {
			dateStr = time.Now().Format("2006-01-02")
		}
	}

	var startDate, endDate time.Time
	var periodDisplay string
	//selectedDate := dateStr

	// Корректируем SelectedDate для поля ввода
	selectedDateForInput := dateStr
	if period == "month" {
		// Для месячного периода добавляем день (первый день месяца)
		selectedDateForInput = dateStr + "-01"
	}

	switch period {
	case "day":
		selected, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.Redirect(http.StatusFound, "/stats?message=Ошибка: Неверный формат даты")
			return
		}
		startDate = selected
		endDate = selected.Add(24 * time.Hour)
		periodDisplay = fmt.Sprintf("%d %s %d", selected.Day(), map[string]string{
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
		}[selected.Month().String()], selected.Year())
	case "week":
		selected, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.Redirect(http.StatusFound, "/stats?message=Ошибка: Неверный формат даты")
			return
		}
		weekday := selected.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		startDate = selected.AddDate(0, 0, -int(weekday-1))
		endDate = startDate.AddDate(0, 0, 7)
		periodDisplay = fmt.Sprintf("с %d %s по %d %s %d", startDate.Day(), map[string]string{
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
		}[startDate.Month().String()], endDate.Day(), map[string]string{
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
		}[endDate.Month().String()], endDate.Year())
	case "month":
		selected, err := time.Parse("2006-01", dateStr)
		if err != nil {
			c.Redirect(http.StatusFound, "/stats?message=Ошибка: Неверный формат месяца")
			return
		}
		startDate = selected
		endDate = selected.AddDate(0, 1, 0)
		periodDisplay = fmt.Sprintf("%s %d", map[string]string{
			"January":   "январь",
			"February":  "февраль",
			"March":     "март",
			"April":     "апрель",
			"May":       "май",
			"June":      "июнь",
			"July":      "июль",
			"August":    "август",
			"September": "сентябрь",
			"October":   "октябрь",
			"November":  "ноябрь",
			"December":  "декабрь",
		}[selected.Month().String()], selected.Year())
	default:
		c.Redirect(http.StatusFound, "/stats?message=Ошибка: Неверный период")
		return
	}

	// Фильтруем транзакции за период
	data := h.financeStore.GetData()
	var filteredTrans []models.Transaction
	for _, t := range data.Transactions {
		if (t.DateTime.Equal(startDate) || t.DateTime.After(startDate)) && (t.DateTime.Before(endDate) || t.DateTime.Equal(endDate)) {
			filteredTrans = append(filteredTrans, t)
		}
	}
	log.Printf("Filtered transactions: %d", len(filteredTrans))
	if len(filteredTrans) == 0 {
		log.Printf("No transactions found between %s and %s", startDate, endDate)
		for _, t := range data.Transactions {
			log.Printf("Transaction: ID=%d, DateTime=%s, Amount=%.2f, IsPositive=%v", t.ID, t.DateTime, t.Amount, t.IsPositive)
		}
	} else {
		for _, t := range filteredTrans {
			log.Printf("Filtered Transaction: ID=%d, DateTime=%s, Amount=%.2f, IsPositive=%v", t.ID, t.DateTime, t.Amount, t.IsPositive)
		}
	}

	// Общая статистика
	totalIncome := 0.0
	totalExpense := 0.0
	for _, t := range filteredTrans {
		if t.IsPositive {
			totalIncome += t.Amount
		} else {
			totalExpense += t.Amount
		}
	}
	netBalance := totalIncome - totalExpense
	daysInPeriod := int(endDate.Sub(startDate).Hours() / 24)
	avgDailyExpense := 0.0
	if daysInPeriod > 0 {
		avgDailyExpense = totalExpense / float64(daysInPeriod)
	}

	// Топ-5 доходов и расходов
	var incomes, expenses []models.Transaction
	for _, t := range filteredTrans {
		if t.IsPositive {
			incomes = append(incomes, t)
		} else {
			expenses = append(expenses, t)
		}
	}
	sort.Slice(incomes, func(i, j int) bool { return incomes[i].Amount > incomes[j].Amount })
	sort.Slice(expenses, func(i, j int) bool { return expenses[i].Amount > expenses[j].Amount })
	topIncomes := []gin.H{}
	topExpenses := []gin.H{}
	for i, t := range incomes {
		if i >= 5 {
			break
		}
		topIncomes = append(topIncomes, gin.H{
			"Amount":      fmt.Sprintf("%.2f", t.Amount),
			"Description": t.Description,
			"DateTime":    t.DateTime.Format("02.01.2006"),
		})
	}
	for i, t := range expenses {
		if i >= 5 {
			break
		}
		topExpenses = append(topExpenses, gin.H{
			"Amount":      fmt.Sprintf("%.2f", t.Amount),
			"Description": t.Description,
			"DateTime":    t.DateTime.Format("02.01.2006"),
		})
	}
	log.Printf("Top Incomes: %d, Top Expenses: %d", len(topIncomes), len(topExpenses))

	// Данные для графика
	type ChartData struct {
		Labels   []string  `json:"labels"`
		Incomes  []float64 `json:"incomes"`
		Expenses []float64 `json:"expenses"`
	}
	chartData := ChartData{}
	// Создаём метки для всего периода
	if period == "month" {
		currentDate := startDate
		for !currentDate.After(endDate) {
			dateStr := currentDate.Format("2006-01-02")
			chartData.Labels = append(chartData.Labels, dateStr)
			chartData.Incomes = append(chartData.Incomes, 0.0)
			chartData.Expenses = append(chartData.Expenses, 0.0)
			currentDate = currentDate.AddDate(0, 0, 1)
		}
	} else if period == "week" {
		currentDate := startDate
		for !currentDate.After(endDate) {
			dateStr := currentDate.Format("2006-01-02")
			chartData.Labels = append(chartData.Labels, dateStr)
			chartData.Incomes = append(chartData.Incomes, 0.0)
			chartData.Expenses = append(chartData.Expenses, 0.0)
			currentDate = currentDate.AddDate(0, 0, 1)
		}
	} else {
		// Для периода "day" используем только дату
		dateStr := startDate.Format("2006-01-02")
		chartData.Labels = append(chartData.Labels, dateStr)
		chartData.Incomes = append(chartData.Incomes, 0.0)
		chartData.Expenses = append(chartData.Expenses, 0.0)
	}

	// Заполняем данные из транзакций
	for _, t := range filteredTrans {
		var dateStr string
		var index int
		var found bool
		if period == "month" || period == "week" || period == "day" {
			// Приводим дату транзакции к формату YYYY-MM-DD
			dateStr = t.DateTime.Format("2006-01-02")
			for i, label := range chartData.Labels {
				if label == dateStr {
					index = i
					found = true
					break
				}
			}
		}
		if found {
			if t.IsPositive {
				chartData.Incomes[index] += t.Amount
				log.Printf("Added Income: Date=%s, Index=%d, Amount=%.2f", dateStr, index, t.Amount)
			} else {
				chartData.Expenses[index] += t.Amount
				log.Printf("Added Expense: Date=%s, Index=%d, Amount=%.2f", dateStr, index, t.Amount)
			}
		} else {
			log.Printf("Transaction not added to chart: Date=%s, Expected Label=%s, Amount=%.2f, IsPositive=%v", t.DateTime, dateStr, t.Amount, t.IsPositive)
			log.Printf("Available Labels: %v", chartData.Labels)
		}
	}
	log.Printf("Chart Data: Labels=%v, Incomes=%v, Expenses=%v", chartData.Labels, chartData.Incomes, chartData.Expenses)

	// Инсайты
	insights := []string{}
	if totalExpense > totalIncome*0.8 {
		insights = append(insights, "Вы потратили более 80% доходов за период. Попробуйте сократить мелкие траты.")
	}
	log.Printf("Insights: %v", insights)

	// Сериализуем ChartData в JSON
	chartDataJSON, err := json.Marshal(chartData)
	if err != nil {
		log.Printf("Error marshaling ChartData: %v", err)
		c.Redirect(http.StatusFound, "/stats?message=Ошибка при формировании данных графика")
		return
	}

	// Передаём числовые значения как float64 и ChartData как JSON-строку
	c.HTML(http.StatusOK, "stats.html", gin.H{
		"SelectedPeriod":  period,
		"SelectedDate":    selectedDateForInput,
		"Period":          periodDisplay,
		"TotalIncome":     totalIncome,
		"TotalExpense":    totalExpense,
		"NetBalance":      netBalance,
		"AvgDailyExpense": avgDailyExpense,
		"TopIncomes":      topIncomes,
		"TopExpenses":     topExpenses,
		"ChartDataJSON":   string(chartDataJSON),
		"Insights":        insights,
	})
}
