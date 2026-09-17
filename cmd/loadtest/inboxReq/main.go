package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// setRandomCacheControl устанавливает случайный заголовок Cache-Control:
// либо "no-store", либо "public, max-age=<случайное время в секундах>"
func setRandomCacheControl(w http.ResponseWriter) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// С вероятностью ~30% отдаём no-store, иначе — случайный max-age
	if r.Intn(10) < 3 {
		w.Header().Set("Cache-Control", "no-store")
		log.Println("Cache-Control: no-store")
		return
	}

	// Случайное время жизни: от 5 секунд до 1 часа (3600 сек)
	maxAge := r.Intn(3600-5+1) + 5
	cacheControl := fmt.Sprintf("public, max-age=%d", maxAge)
	w.Header().Set("Cache-Control", cacheControl)
	log.Printf("Cache-Control: %s", cacheControl)
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Логируем входящий запрос
	log.Printf("=== Входящий запрос: %s %s от %s ===", r.Method, r.URL.Path, r.RemoteAddr)

	// Устанавливаем случайный Cache-Control
	setRandomCacheControl(w)

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
	// Инициализируем глобальный источник случайности
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/", handler)

	addr := ":8081"
	log.Printf("Сервер запущен на http://localhost%s", addr)
	log.Println("Пример: http://localhost:8081/?name=ivan&age=25&city=msk")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}