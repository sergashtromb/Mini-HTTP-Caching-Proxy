// this file is needed to implement the incoming request handler

package inboxhandler

import (
	"context"
	"strings"
	"sync"
	"time"
	"log/slog"
	"mini_http_caching_proxy/tools"
	"net"
	"net/http"
)

type TranspanentProxy struct {
	buff 		sync.Pool //for copy data in https connect
	connManager *ConnManager
}


func NewTranspanentProxy(memBuff int, cm *ConnManager) *TranspanentProxy {
	return &TranspanentProxy{
		buff: sync.Pool{
			New: func() interface{} {
				bf := make([]byte, memBuff*1024)
				return &bf
			},
		},
		connManager: cm,
	}
}

func (ih *TranspanentProxy) InboxRequest(w http.ResponseWriter, r *http.Request) {
	sendHttpRequest(w, r)
}

func (ih *TranspanentProxy) HandleConnection(w http.ResponseWriter, r *http.Request) {
	
	hijack, ok := w.(http.Hijacker)
	if !ok {
		slog.Error("Hijacker don't supported")
		http.Error(w, "Hijacker don't supported", http.StatusServiceUnavailable)
		return
	}

	target, err := createTarget(r.Context(), r.Host, 10*time.Second)
	if err != nil {
		slog.Error("Failed to connection to the target resource", "err", err)
		http.Error(w, "Server error", http.StatusServiceUnavailable)

		if conn, _, err := hijack.Hijack(); err == nil {
			err := tools.SendBadGatterway(conn)
			if err != nil {
				slog.Error("Failed send Bad Gatterway", "err", err)
			}
			conn.Close()
		}

		return
	}

	clientConn, _, err := hijack.Hijack()
	if err != nil {
		slog.Error("Error hijack", "err", err)
		http.Error(w, "Server error", http.StatusServiceUnavailable)
		target.Close()
		return
	}

	err = tools.SendSuccesConnection(clientConn)
	if err != nil {
		slog.Error("Failed send success connection", "err", err)
		target.Close()
		clientConn.Close()
		return
	}

	ih.connManager.Register(target, clientConn)
}

func (ih *TranspanentProxy) Shutdown(ctx context.Context) error {

	if err := ih.connManager.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

func createTarget(ctx context.Context, host string, timeout time.Duration) (net.Conn, error) {

	dialer := &net.Dialer{
		FallbackDelay: 300*time.Millisecond,
		Timeout: timeout,
	}	

	target, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		slog.Error("Failed connect to host", "err", err)
		return nil, err
	}

	return target, nil
}

func generateKey(r *http.Request) string {

	var strBuilder strings.Builder
	strBuilder.WriteString(r.Method)
	strBuilder.WriteString(r.Host)
	strBuilder.WriteString(r.URL.Path)

	return strBuilder.String()
}

