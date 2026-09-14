package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Конфигурация
const (
	dirPath      = "C:\\Temp\\proxy"             // Путь к отслеживаемой директории (сейчас установлена текущая папка)
	scanInterval = 20 * time.Nanosecond // Интервал проверки директории
)

func main() {
	fmt.Printf("Запущено отслеживание директории: %s\n", dirPath)
	fmt.Printf("Интервал проверки: %v. Для выхода нажмите Ctrl+C\n\n", scanInterval)

	// Настройка graceful shutdown для корректного выхода по Ctrl+C
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	// Первая проверка сразу при запуске
	checkAndPrintFiles(dirPath)

	for {
		select {
		case <-ticker.C:
			checkAndPrintFiles(dirPath)
		case <-ctx.Done():
			fmt.Println("\nОтслеживание остановлено.")
			return
		}
	}
}

// checkAndPrintFiles считает файлы и выводит результат
func checkAndPrintFiles(path string) {
	count, err := countFiles(path)
	if err != nil {
		log.Printf("Ошибка при чтении директории: %v\n", err)
		return
	}
	if count > 1 {
		currentTime := time.Now().Format("15:04:05")
		fmt.Printf("[%s] Количество файлов в директории: %d\n", currentTime, count)
	}
	
}

// countFiles считает только файлы в указанной папке (не включая подпапки)
func countFiles(path string) (int, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}

	fileCount := 0
	for _, entry := range entries {
		// Проверяем, что это файл, а не поддиректория
		if !entry.IsDir() {
			fileCount++
		}
	}

	return fileCount, nil
}
