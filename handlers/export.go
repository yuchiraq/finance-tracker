// handlers/export.go
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

type ExportHandler struct {
	workLogStore *storage.WorkLogStorage
}

func NewExportHandler(workLogStore *storage.WorkLogStorage) *ExportHandler {
	return &ExportHandler{workLogStore: workLogStore}
}

func (h *ExportHandler) ExportWorkLogPDF(c *gin.Context) {
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
	var allWorkedHours, overHours float64

	for _, entry := range data.Entries {
		entryDate, err := time.Parse("2006-01-02", entry.Date)
		if err != nil {
			continue
		}
		if entryDate.Year() == selectedMonth.Year() && entryDate.Month() == selectedMonth.Month() && !entry.IsDayOff {
			filteredEntries = append(filteredEntries, entry)

			start, _ := time.Parse("15:04", entry.StartTime)
			end, _ := time.Parse("15:04", entry.EndTime)
			duration := end.Sub(start).Hours()
			if duration < 0 {
				duration += 24
			}
			allWorkedHours += duration
			if duration > 8 {
				overHours += duration - 8
			}
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
	pdf.CellFormat(20, 10, "Число", "1", 0, "C", true, 0, "")
	pdf.CellFormat(50, 10, "Место", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 10, "Время", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 10, "Длительность", "1", 0, "C", true, 0, "")
	pdf.Ln(-1)

	pdf.SetFont("DejaVu", "", 10)
	pdf.SetFillColor(255, 255, 255)
	for _, entry := range filteredEntries {
		date, _ := time.Parse("2006-01-02", entry.Date)
		formattedDate := fmt.Sprintf("%02d", date.Day())

		start, _ := time.Parse("15:04", entry.StartTime)
		end, _ := time.Parse("15:04", entry.EndTime)
		duration := end.Sub(start).Hours()
		if duration < 0 {
			duration += 24
		}
		hoursWorked := fmt.Sprintf("%.1f ч", duration)

		pdf.CellFormat(20, 8, formattedDate, "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 8, entry.Place, "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, 8, fmt.Sprintf("%s - %s", entry.StartTime, entry.EndTime), "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 8, hoursWorked, "1", 0, "C", false, 0, "")
		pdf.Ln(-1)
	}

	pdf.Ln(5)
	pdf.SetFont("DejaVu", "", 12)
	pdf.Cell(0, 10, fmt.Sprintf("Всего рабочих дней: %d", len(filteredEntries)))
	pdf.Ln(5)
	pdf.Cell(0, 10, fmt.Sprintf("Всего отработано часов: %.1f", allWorkedHours))
	pdf.Ln(5)
	pdf.Cell(0, 10, fmt.Sprintf("Отработано часов выше нормы (более 8 часов в сутки): %.1f", overHours))
	pdf.Ln(5)
	pdf.Cell(0, 10, fmt.Sprintf("Всего часов (включая сверхурочные): %.1f", allWorkedHours+overHours))

	fileName := fmt.Sprintf("worklog_%s.pdf", selectedMonth.Format("2006-01"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Type", "application/pdf")

	err = pdf.Output(c.Writer)
	if err != nil {
		c.Redirect(http.StatusFound, "/worklog?message=Ошибка при генерации PDF")
		return
	}
}
