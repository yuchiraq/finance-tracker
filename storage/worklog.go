// storage/worklog.go
package storage

import (
	"encoding/json"
	"finance-tracker/models"
	"fmt"
	"os"
	"sync"
)

type WorkLogStorage struct {
	data     models.WorkLogData
	filePath string
	mutex    sync.Mutex
}

func NewWorkLogStorage(filePath string) *WorkLogStorage {
	return &WorkLogStorage{
		filePath: filePath,
		data: models.WorkLogData{
			Entries: []models.WorkEntry{},
		},
	}
}

func (s *WorkLogStorage) Load() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	fmt.Println("Загрузка данных табеля из файла:", s.filePath)

	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		fmt.Println("Файл табеля не существует, создаём новый")
		return nil
	}

	fileData, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("ошибка при чтении файла: %v", err)
	}

	if err := json.Unmarshal(fileData, &s.data); err != nil {
		return fmt.Errorf("ошибка при декодировании JSON: %v", err)
	}

	fmt.Printf("Загруженные записи табеля: %d\n", len(s.data.Entries))
	return nil
}

func (s *WorkLogStorage) Save() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	fmt.Println("Сохранение данных табеля в файл:", s.filePath)

	fileData, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка при кодировании в JSON: %v", err)
	}

	if err := os.WriteFile(s.filePath, fileData, 0644); err != nil {
		return fmt.Errorf("ошибка при записи в файл: %v", err)
	}

	fmt.Println("Данные табеля успешно сохранены")
	return nil
}

func (s *WorkLogStorage) GetData() *models.WorkLogData {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return &s.data
}
