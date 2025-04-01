// main.go
package main

import (
	"finance-tracker/auth"
	"finance-tracker/handlers"
	"finance-tracker/storage"
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("Запуск приложения...")

	// Инициализация хранилищ
	financeStore := storage.NewFinanceStorage("finance_data.json")
	workLogStore := storage.NewWorkLogStorage("worklog_data.json")

	// Загружаем данные
	if err := financeStore.Load(); err != nil {
		fmt.Println("Ошибка загрузки финансовых данных:", err)
	}
	if err := workLogStore.Load(); err != nil {
		fmt.Println("Ошибка загрузки данных табеля:", err)
	}

	// Пересчитываем баланс
	financeStore.RecalculateBalances()

	// Настройка Gin
	r := gin.Default()

	// Middleware авторизации
	r.Use(auth.Middleware())

	// Статические файлы
	r.Static("/static", "./static")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.StaticFile("/apple-touch-icon-precomposed.png", "./static/apple-touch-icon-precomposed.png")
	r.StaticFile("/apple-touch-icon.png", "./static/apple-touch-icon.png")

	// Загрузка шаблонов
	r.LoadHTMLGlob("templates/*")

	// Регистрация маршрутов
	handlers.RegisterRoutes(r, financeStore, workLogStore)

	// Запуск сервера
	fmt.Println("Сервер запущен на http://localhost:8088")
	if err := r.Run(":8088"); err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
	}
}
