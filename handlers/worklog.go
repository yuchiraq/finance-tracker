package handlers

import (
	"finance-tracker/models"
	"finance-tracker/storage"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

type WorkLogHandler struct {
	workLogStore *storage.WorkLogStorage
}

func NewWorkLogHandler(workLogStore *storage.WorkLogStorage) *WorkLogHandler {
	return &WorkLogHandler{workLogStore: workLogStore}
}

// WorkLogSummary представляет данные о работе за месяц
type WorkLogSummary struct {
	WorkDays          int     `json:"work_days"`
	TotalHours        float64 `json:"total_hours"`
	OvertimeHours     float64 `json:"overtime_hours"`
	TotalWithOvertime float64 `json:"total_with_overtime"`
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
			if duration > 7 {
				// Время обеда
				hoursWorked = fmt.Sprintf("%.1f часов - 1", duration)
			} else {
				hoursWorked = fmt.Sprintf("%.1f часов", duration)
			}
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

func (h *WorkLogHandler) GetWorkLogSummary(c *gin.Context) {
	month := c.Query("month") // Формат: "YYYY-MM"
	if month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Месяц не указан"})
		return
	}

	monthTime, err := time.Parse("2006-01", month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат месяца"})
		return
	}

	data := h.workLogStore.GetData()
	var totalHours, overtimeHours float64
	var workDays int

	for _, entry := range data.Entries {
		entryDate, _ := time.Parse("2006-01-02", entry.Date)
		if entryDate.Year() == monthTime.Year() && entryDate.Month() == monthTime.Month() && !entry.IsDayOff {
			workDays++
			start, _ := time.Parse("15:04", entry.StartTime)
			end, _ := time.Parse("15:04", entry.EndTime)
			duration := end.Sub(start).Hours()
			if duration < 0 {
				duration += 24
			}
			// Учитываем обед, если работа больше 7 часов
			if duration > 7 {
				duration -= 1 // Вычитаем час на обед
			}
			if duration > 8 {
				overtimeHours += duration - 8
			}
			totalHours += duration
		}
	}

	summary := WorkLogSummary{
		WorkDays:          workDays,
		TotalHours:        totalHours,
		OvertimeHours:     overtimeHours,
		TotalWithOvertime: totalHours + overtimeHours,
	}

	c.JSON(http.StatusOK, summary)
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
		startTime = "08:00"
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

func (h *WorkLogHandler) EditWork(c *gin.Context) {
	date := c.Param("date")
	place := c.PostForm("place")
	startTime := c.PostForm("start_time")
	endTime := c.PostForm("end_time")
	isDayOff := c.PostForm("is_day_off") == "on"

	data := h.workLogStore.GetData()
	for i, entry := range data.Entries {
		if entry.Date == date {
			if !isDayOff {
				if place == "" {
					c.Redirect(http.StatusFound, "/worklog?message=Ошибка: Укажите место работы")
					return
				}
				if startTime == "" || endTime == "" {
					c.Redirect(http.StatusFound, "/worklog?message=Ошибка: Укажите время работы")
					return
				}
				_, err := time.Parse("15:04", startTime)
				if err != nil {
					c.Redirect(http.StatusFound, "/worklog?message=Ошибка: Неверный формат времени начала")
					return
				}
				_, err = time.Parse("15:04", endTime)
				if err != nil {
					c.Redirect(http.StatusFound, "/worklog?message=Ошибка: Неверный формат времени окончания")
					return
				}
			} else {
				place = ""
				startTime = "08:00"
				endTime = "17:00"
			}

			data.Entries[i].Place = place
			data.Entries[i].StartTime = startTime
			data.Entries[i].EndTime = endTime
			data.Entries[i].IsDayOff = isDayOff
			break
		}
	}

	if err := h.workLogStore.Save(); err != nil {
		c.Redirect(http.StatusFound, "/worklog?message=Ошибка при сохранении данных")
		return
	}

	c.Redirect(http.StatusFound, "/worklog?message=Запись о работе обновлена")
}
