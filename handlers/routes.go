// handlers/routes.go
package handlers

import (
	"finance-tracker/storage"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, financeStore *storage.FinanceStorage, workLogStore *storage.WorkLogStorage) {
	financeHandler := NewFinanceHandler(financeStore, workLogStore)
	workLogHandler := NewWorkLogHandler(workLogStore)

	// Маршруты для финансов
	r.GET("/", financeHandler.Index)
	r.POST("/add", financeHandler.AddTransaction)
	r.POST("/edit/:id", financeHandler.EditTransaction)
	r.POST("/delete/:id", financeHandler.DeleteTransaction)
	r.GET("/api/transactions", financeHandler.GetTransactions)

	// Маршруты для табеля
	r.GET("/worklog", workLogHandler.WorkLog)
	r.POST("/add-work", workLogHandler.AddWork)
	r.GET("/worklog/export", workLogHandler.ExportWorkLogPDF) // Новый маршрут для экспорта
}
