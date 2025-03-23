package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"finance-tracker/handlers"
	"finance-tracker/models"

	"github.com/gin-gonic/gin"
)

var data models.FinanceData
var dataFile = "finance_data.json"
var dataMutex sync.Mutex

func main() {
	fmt.Println("Запуск приложения...")

	// Загружаем существующие данные
	loadData()

	// Пересчитываем баланс
	recalculateBalances()

	// Сохраняем исправленные данные
	saveData()

	r := gin.Default()

	// Добавляем parseFloat в шаблоны для проверки отрицательного баланса
	r.SetFuncMap(template.FuncMap{
		"parseFloat": func(s string) float64 {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		},
	})

	// Статические файлы (CSS, JS)
	r.Static("/static", "./static")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.StaticFile("/apple-touch-icon-precomposed.png", "./static/apple-touch-icon-precomposed.png")
	r.StaticFile("/apple-touch-icon.png", "./static/apple-touch-icon.png")

	// Загрузка шаблонов
	r.LoadHTMLGlob("templates/*")

	// Передаём данные и функцию сохранения в handlers
	handlers.Init(&data, saveData)

	// Маршруты
	r.GET("/", handlers.Index)
	r.POST("/add", handlers.AddTransaction)
	r.POST("/edit/:id", handlers.EditTransaction)
	r.POST("/delete/:id", handlers.DeleteTransaction)

	// Запуск сервера
	fmt.Println("Сервер запущен на http://localhost:8088")
	if err := r.Run(":8088"); err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
	}
}

// Временная структура для обратной совместимости со старым форматом данных
type legacyFinanceData struct {
	Transactions []models.Transaction `json:"Transactions"`
	Balance      float64              `json:"Balance"` // Старое поле
	Balances     map[string]float64   `json:"Balances"`
}

// Загрузка данных из файла
func loadData() {
	dataMutex.Lock()
	defer dataMutex.Unlock()

	fmt.Println("Загрузка данных из файла:", dataFile)

	// Проверяем существование директории
	dir := filepath.Dir(dataFile)
	if dir != "" {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Println("Директория не существует, создаём:", dir)
			os.MkdirAll(dir, 0755)
		}
	}

	// Проверяем существование файла
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		fmt.Println("Файл данных не существует, создаём новый")
		// Создаём пустой файл с начальными данными
		data = models.FinanceData{
			Transactions: []models.Transaction{},
			Balances:     make(map[string]float64),
		}
		return
	}

	// Читаем данные из файла
	fileData, err := os.ReadFile(dataFile)
	if err != nil {
		fmt.Println("Ошибка при чтении файла:", err)
		return
	}

	fmt.Println("Содержимое файла:", string(fileData))

	// Декодируем JSON с обработкой старого формата
	var rawData legacyFinanceData
	err = json.Unmarshal(fileData, &rawData)
	if err != nil {
		fmt.Println("Ошибка при декодировании JSON:", err)
		return
	}

	// Инициализируем Balances, если его нет
	if rawData.Balances == nil {
		rawData.Balances = make(map[string]float64)
		fmt.Println("Balances не инициализирован, создаём пустую карту")
	}

	// Переносим старое поле Balance в Balances["BYN"], если оно есть
	if rawData.Balance != 0 && rawData.Balances["BYN"] == 0 {
		rawData.Balances["BYN"] = rawData.Balance
		fmt.Println("Переносим Balance в Balances[BYN]:", rawData.Balance)
	}

	// Исправляем транзакции
	for i, t := range rawData.Transactions {
		// Если amount отрицательный, делаем его положительным
		if t.Amount < 0 {
			fmt.Printf("Исправляем отрицательный amount в транзакции %d: %f → %f\n", t.ID, t.Amount, -t.Amount)
			rawData.Transactions[i].Amount = -t.Amount
		}
		// Заполняем отсутствующие поля
		if t.Currency == "" {
			rawData.Transactions[i].Currency = "BYN"
			fmt.Printf("Устанавливаем валюту по умолчанию (BYN) для транзакции %d\n", t.ID)
		}
		if t.Category == "" {
			rawData.Transactions[i].Category = "Другое"
			fmt.Printf("Устанавливаем категорию по умолчанию (Другое) для транзакции %d\n", t.ID)
		}
		// Исправляем IsPositive для транзакций, которые выглядят как доходы
		if strings.ToLower(t.Description) == "зп" || strings.ToLower(t.Category) == "зарплата" {
			if !t.IsPositive {
				fmt.Printf("Исправляем IsPositive для транзакции %d (%s): false → true\n", t.ID, t.Description)
				rawData.Transactions[i].IsPositive = true
				// Также исправляем категорию, если это "зп"
				if strings.ToLower(t.Description) == "зп" && t.Category != "Зарплата" {
					rawData.Transactions[i].Category = "Зарплата"
					fmt.Printf("Исправляем категорию для транзакции %d: %s → Зарплата\n", t.ID, t.Category)
				}
			}
		}
	}

	// Преобразуем в новую структуру
	data = models.FinanceData{
		Transactions: rawData.Transactions,
		Balances:     rawData.Balances,
	}

	// Логируем загруженные данные
	fmt.Printf("Загруженные транзакции: %+v\n", data.Transactions)
	fmt.Printf("Загруженный баланс: %+v\n", data.Balances)
}

// Пересчёт баланса
func recalculateBalances() {
	dataMutex.Lock()
	defer dataMutex.Unlock()

	fmt.Println("Пересчёт баланса...")

	// Сбрасываем Balances
	data.Balances = make(map[string]float64)

	// Пересчитываем баланс на основе транзакций
	for _, t := range data.Transactions {
		amount := t.Amount
		if amount < 0 {
			amount = -amount // Убедимся, что amount положительный
			fmt.Printf("Обнаружен отрицательный amount в транзакции %d, исправляем: %f\n", t.ID, t.Amount)
		}
		if t.IsPositive {
			data.Balances[t.Currency] += amount
			fmt.Printf("Добавляем доход %f %s (транзакция %d)\n", amount, t.Currency, t.ID)
		} else {
			data.Balances[t.Currency] -= amount
			fmt.Printf("Вычитаем расход %f %s (транзакция %d)\n", amount, t.Currency, t.ID)
		}
	}

	// Очищаем нулевые балансы
	for currency, balance := range data.Balances {
		if balance == 0 {
			fmt.Printf("Удаляем нулевой баланс для валюты %s\n", currency)
			delete(data.Balances, currency)
		}
	}

	// Логируем пересчитанный баланс
	fmt.Printf("Пересчитанный баланс: %+v\n", data.Balances)
}

// Сохранение данных в файл
func saveData() {
	dataMutex.Lock()
	defer dataMutex.Unlock()

	fmt.Println("Сохранение данных в файл:", dataFile)

	fileData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Ошибка при кодировании в JSON:", err)
		return
	}

	err = os.WriteFile(dataFile, fileData, 0644)
	if err != nil {
		fmt.Println("Ошибка при записи в файл:", err)
	} else {
		fmt.Println("Данные успешно сохранены")
	}
}
