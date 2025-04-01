// storage/finance.go
package storage

import (
	"encoding/json"
	"finance-tracker/models"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type FinanceStorage struct {
	data     models.FinanceData
	filePath string
	mutex    sync.Mutex
}

func NewFinanceStorage(filePath string) *FinanceStorage {
	return &FinanceStorage{
		filePath: filePath,
		data: models.FinanceData{
			Transactions: []models.Transaction{},
			Balances:     make(map[string]float64),
		},
	}
}

func (s *FinanceStorage) Load() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	fmt.Println("Загрузка финансовых данных из файла:", s.filePath)

	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		fmt.Println("Файл финансовых данных не существует, создаём новый")
		return nil
	}

	fileData, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("ошибка при чтении файла: %v", err)
	}

	if err := json.Unmarshal(fileData, &s.data); err != nil {
		return fmt.Errorf("ошибка при декодировании JSON: %v", err)
	}

	fmt.Printf("Загруженные транзакции: %d\n", len(s.data.Transactions))
	return nil
}

func (s *FinanceStorage) Save() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	fmt.Println("Сохранение финансовых данных в файл:", s.filePath)

	// Создаём резервную копию
	s.createBackup()

	fileData, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка при кодировании в JSON: %v", err)
	}

	if err := os.WriteFile(s.filePath, fileData, 0644); err != nil {
		return fmt.Errorf("ошибка при записи в файл: %v", err)
	}

	fmt.Println("Финансовые данные успешно сохранены")
	return nil
}

func (s *FinanceStorage) createBackup() {
	const maxBackups = 10
	backupDir := "backups"
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		os.Mkdir(backupDir, 0755)
	}

	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return
	}

	currentData, err := os.ReadFile(s.filePath)
	if err != nil {
		fmt.Println("Ошибка чтения файла для резервного копирования:", err)
		return
	}

	backupFiles, err := filepath.Glob(filepath.Join(backupDir, "finance_data_backup_*.json"))
	if err != nil {
		fmt.Println("Ошибка получения списка резервных копий:", err)
		return
	}

	sort.Slice(backupFiles, func(i, j int) bool {
		infoI, _ := os.Stat(backupFiles[i])
		infoJ, _ := os.Stat(backupFiles[j])
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	for len(backupFiles) >= maxBackups {
		os.Remove(backupFiles[0])
		backupFiles = backupFiles[1:]
	}

	backupFile := filepath.Join(backupDir, fmt.Sprintf("finance_data_backup_%d.json", time.Now().Unix()))
	if err := os.WriteFile(backupFile, currentData, 0644); err != nil {
		fmt.Println("Ошибка создания резервной копии:", err)
	} else {
		fmt.Println("Резервная копия создана:", backupFile)
	}
}

func (s *FinanceStorage) GetData() *models.FinanceData {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return &s.data
}

func (s *FinanceStorage) RecalculateBalances() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	fmt.Println("Пересчёт баланса...")

	// Сбрасываем баланс
	s.data.Balances = make(map[string]float64)

	// Пересчитываем баланс для каждой валюты
	for _, t := range s.data.Transactions {
		if t.IsPositive {
			s.data.Balances[t.Currency] += t.Amount
		} else {
			s.data.Balances[t.Currency] -= t.Amount
		}
	}

	fmt.Println("Баланс пересчитан:", s.data.Balances)
}
