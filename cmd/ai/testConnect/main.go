package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

func handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[CONNECT] Браузер запрашивает туннель к: %s", r.Host)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Println("[ERROR] Сервер не поддерживает Hijack")
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	// Важно: подключаемся принудительно через "tcp4" для обхода проблем с IPv6
	destConn, err := net.DialTimeout("tcp4", r.Host, 10*time.Second)
	if err != nil {
		log.Printf("[ERROR] Не удалось подключиться к целевому сайту %s: %v", r.Host, err)
		// Перехватываем соединение, чтобы корректно ответить браузеру об ошибке
		if clientConn, _, errH := hijacker.Hijack(); errH == nil {
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			clientConn.Close()
		}
		return
	}
	defer destConn.Close()

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		log.Printf("[ERROR] Ошибка Hijack соединения клиента: %v", err)
		return
	}
	defer clientConn.Close()

	// Отправляем строго то, что ждет Chrome
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		log.Printf("[ERROR] Не удалось отправить статус клиенту: %v", err)
		return
	}

	log.Printf("[SUCCESS] Туннель для %s успешно создан. Начинаем обмен данными.", r.Host)

	chDone := make(chan bool, 2)
	go func() {
		io.Copy(destConn, clientConn)
		chDone <- true
	}()
	go func() {
		io.Copy(clientConn, destConn)
		chDone <- true
	}()

	<-chDone
	log.Printf("[CLOSED] Туннель для %s закрыт.", r.Host)
}

func main() {
	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: http.HandlerFunc(handleConnect),
	}
	log.Println("Сервер запущен на http://127.0.0.1:8080. Ожидание запросов...")
	log.Fatal(server.ListenAndServe())
}
