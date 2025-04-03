// handlers/routes.go
package handlers

import (
	"finance-tracker/storage"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, financeStore *storage.FinanceStorage, workLogStore *storage.WorkLogStorage) {
	financeHandler := NewFinanceHandler(financeStore, workLogStore)
	workLogHandler := NewWorkLogHandler(workLogStore)
	statsHandler := NewStatsHandler(financeStore)
	exportHandler := NewExportHandler(workLogStore)

	// Маршруты для финансов
	r.GET("/", financeHandler.Index)
	r.POST("/add", financeHandler.AddTransaction)
	r.POST("/edit/:id", financeHandler.EditTransaction)
	r.POST("/delete/:id", financeHandler.DeleteTransaction)
	r.GET("/api/transactions", financeHandler.GetTransactions)

	// Маршруты для табеля
	r.GET("/worklog", workLogHandler.WorkLog)
	r.POST("/add-work", workLogHandler.AddWork)
	r.POST("/edit-work/:date", workLogHandler.EditWork) // Новый маршрут для редактирования
	r.GET("/worklog/export", exportHandler.ExportWorkLogPDF)

	// Маршруты для статистики
	r.GET("/stats", statsHandler.Stats)
}
