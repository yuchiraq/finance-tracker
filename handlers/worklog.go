// handlers/worklog.go
package handlers

import (
	"finance-tracker/models"
	"finance-tracker/storage"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
)

type WorkLogHandler struct {
	workLogStore *storage.WorkLogStorage
}

func NewWorkLogHandler(workLogStore *storage.WorkLogStorage) *WorkLogHandler {
	return &WorkLogHandler{workLogStore: workLogStore}
}

func (h *WorkLogHandler) WorkLog(c *gin.Context) {
	data := h.workLogStore.GetData()
	formattedEntries := []gin.H{}
	for _, entry := range data.Entries {
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
				duration += 24
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

	sort.Slice(formattedEntries, func(i, j int) bool {
		return formattedEntries[i]["Date"].(string) > formattedEntries[j]["Date"].(string)
	})

	c.HTML(http.StatusOK, "worklog.html", gin.H{
		"entries": formattedEntries,
	})
}

func (h *WorkLogHandler) AddWork(c *gin.Context) {
	place := c.PostForm("place")
	startTime := c.PostForm("start_time")
	endTime := c.PostForm("end_time")
	isDayOff := c.PostForm("is_day_off") == "true"

	today := time.Now().Format("2006-01-02")

	data := h.workLogStore.GetData()
	for _, entry := range data.Entries {
		if entry.Date == today {
			c.Redirect(http.StatusFound, "/?message=Запись за сегодня уже существует")
			return
		}
	}

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
	} else {
		place = ""
		startTime = "09:00"
		endTime = "17:00"
	}

	newEntry := models.WorkEntry{
		Date:      today,
		Place:     place,
		StartTime: startTime,
		EndTime:   endTime,
		IsDayOff:  isDayOff,
	}
	data.Entries = append(data.Entries, newEntry)

	if err := h.workLogStore.Save(); err != nil {
		c.Redirect(http.StatusFound, "/?message=Ошибка при сохранении данных")
		return
	}

	c.Redirect(http.StatusFound, "/?message=Запись о работе добавлена")
}

func (h *WorkLogHandler) ExportWorkLogPDF(c *gin.Context) {
	monthStr := c.Query("month")
	if monthStr == "" {
		c.Redirect(http.StatusFound, "/worklog?message=Ошибка: Укажите месяц для экспорта")
		return
	}

	selectedMonth, err := time.Parse("2006-01", monthStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/worklog?message=Ошибка: Неверный формат месяца")
		return
	}

	data := h.workLogStore.GetData()
	var filteredEntries []models.WorkEntry
	var allWorkedHours float64

	for _, entry := range data.Entries {
		entryDate, err := time.Parse("2006-01-02", entry.Date)
		if err != nil {
			continue
		}
		if entryDate.Year() == selectedMonth.Year() && entryDate.Month() == selectedMonth.Month() && !entry.IsDayOff {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	sort.Slice(filteredEntries, func(i, j int) bool {
		dateI, _ := time.Parse("2006-01-02", filteredEntries[i].Date)
		dateJ, _ := time.Parse("2006-01-02", filteredEntries[j].Date)
		return dateI.Before(dateJ)
	})

	if len(filteredEntries) == 0 {
		c.Redirect(http.StatusFound, "/worklog?message=Нет рабочих дней за выбранный месяц")
		return
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.AddUTF8Font("DejaVu", "", "DejaVuSans.ttf")
	pdf.SetFont("DejaVu", "", 16)

	monthName := map[time.Month]string{
		time.January:   "Январь",
		time.February:  "Февраль",
		time.March:     "Март",
		time.April:     "Апрель",
		time.May:       "Май",
		time.June:      "Июнь",
		time.July:      "Июль",
		time.August:    "Август",
		time.September: "Сентябрь",
		time.October:   "Октябрь",
		time.November:  "Ноябрь",
		time.December:  "Декабрь",
	}[selectedMonth.Month()]
	pdf.Cell(0, 10, fmt.Sprintf("Табель работ за %s %d", monthName, selectedMonth.Year()))
	pdf.Ln(15)

	pdf.SetFont("DejaVu", "", 12)
	pdf.SetFillColor(200, 200, 200)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(20, 10, "Дата", "1", 0, "C", true, 0, "")
	pdf.CellFormat(50, 10, "Место", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 10, "Время", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 10, "Длительность", "1", 0, "C", true, 0, "")
	pdf.Ln(-1)

	pdf.SetFont("DejaVu", "", 10)
	pdf.SetFillColor(255, 255, 255)
	for _, entry := range filteredEntries {
		date, _ := time.Parse("2006-01-02", entry.Date)
		formattedDate := fmt.Sprintf("%02d.%02d.%d", date.Day(), date.Month(), date.Year())

		start, _ := time.Parse("15:04", entry.StartTime)
		end, _ := time.Parse("15:04", entry.EndTime)
		duration := end.Sub(start).Hours()
		if duration < 0 {
			duration += 24
		}
		hoursWorked := fmt.Sprintf("%.1f ч", duration)

		allWorkedHours += duration

		pdf.CellFormat(20, 8, formattedDate, "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 8, entry.Place, "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, 8, fmt.Sprintf("%s - %s", entry.StartTime, entry.EndTime), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 8, hoursWorked, "1", 0, "C", false, 0, "")
		pdf.Ln(-1)
	}

	pdf.Ln(5)
	pdf.SetFont("DejaVu", "", 12)
	pdf.Cell(0, 10, fmt.Sprintf("Всего рабочих дней: %d", len(filteredEntries)))
	pdf.Ln(5)
	pdf.Cell(0, 10, fmt.Sprintf("Всего отработано часов: %f", allWorkedHours))

	fileName := fmt.Sprintf("worklog_%s.pdf", selectedMonth.Format("2006-01"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Type", "application/pdf")

	err = pdf.Output(c.Writer)
	if err != nil {
		c.Redirect(http.StatusFound, "/worklog?message=Ошибка при генерации PDF")
		return
	}
}
