package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	// Логируем входящий запрос
	log.Printf("=== Входящий запрос: %s %s от %s ===", r.Method, r.URL.Path, r.RemoteAddr)

	// Собираем все GET-параметры
	query := r.URL.Query()

	if len(query) == 0 {
		log.Println("Параметры отсутствуют")
		// Задержка 1 секунда
		time.Sleep(1 * time.Second)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"no query parameters"}`)
		return
	}

	// Выводим все ключи и значения в консоль
	fmt.Println("Полученные GET-параметры:")
	for key, values := range query {
		for _, v := range values {
			fmt.Printf("  %s = %s\n", key, v)
		}
	}

	// Секундная задержка перед ответом
	log.Println("Ожидание 1 секунда перед ответом...")
	time.Sleep(1 * time.Second)

	// Формируем ответ — отправляем те же параметры обратно
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Ручная сборка JSON, чтобы сохранить множественные значения
	fmt.Fprint(w, "{")
	first := true
	for key, values := range query {
		if !first {
			fmt.Fprint(w, ",")
		}
		first = false
		if len(values) == 1 {
			fmt.Fprintf(w, "%q:%q", key, values[0])
		} else {
			// Если ключ повторяется — отдаём массивом
			fmt.Fprintf(w, "%q:[", key)
			for i, v := range values {
				if i > 0 {
					fmt.Fprint(w, ",")
				}
				fmt.Fprintf(w, "%q", v)
			}
			fmt.Fprint(w, "]")
		}
	}
	fmt.Fprint(w, "}")

	log.Println("Ответ отправлен")
}

func main() {
	http.HandleFunc("/", handler)

	addr := ":8081"
	log.Printf("Сервер запущен на http://localhost%s", addr)
	log.Println("Пример: http://localhost:8081/?name=ivan&age=25&city=msk")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}