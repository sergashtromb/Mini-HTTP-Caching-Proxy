// this file is needed to implement the incoming request handler

package inboxhandler

import (
	"bytes"
	"sync"
	"time"

	//"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/tools"
	"net"
	"net/http"
	"slices"
)

type InboxHandler struct {
	client 	http.Client
	cnf 	*config.Config
	buff 	sync.Pool
}


func NewInboxHandler(cnf *config.Config) *InboxHandler {
	return &InboxHandler{
		client: http.Client{},
		cnf: cnf,
		buff: sync.Pool{
			New: func() interface{} {
				bf := make([]byte, cnf.MemBuff*1024)
				return &bf
			},
		},
	}
}

func (ih *InboxHandler) HandleInboxReq(w http.ResponseWriter, r *http.Request) {
	isOurHost := slices.Contains(ih.cnf.Hosts, r.Host)

	if isOurHost {
		ih.workOurHostRequest(w, r)
	} else {
		ih.sendHttpRequest(w, r)
	}
}

func (ih *InboxHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	
	hijack, ok := w.(http.Hijacker)
	if !ok {
		slog.Error("Hijacker don't supported")
		http.Error(w, "Hijacker don't supported", http.StatusServiceUnavailable)
		return
	}

	target, err := createTarget(r.Host, 10*time.Second)
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
	defer target.Close()

	clientConn, _, err := hijack.Hijack()
	if err != nil {
		slog.Error("Error hijack", "err", err)
		http.Error(w, "Server error", http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	err = tools.SendSuccesConnection(clientConn)
	if err != nil {
		slog.Error("Failed send success connection", "err", err)
		return
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		transfer(target, clientConn, &ih.buff)
	})
	wg.Go(func() {
		transfer(clientConn, target, &ih.buff)
	})
	
	wg.Wait()
}

func createTarget(host string, timeout time.Duration) (net.Conn, error) {

	targetIPv6, err := net.DialTimeout("tcp6", host, timeout)
	if err == nil {
		return targetIPv6, nil
	}

	targetIPv4, err := net.DialTimeout("tcp4", host, timeout)
	if err != nil {
		slog.Error("Don't connect IPv4", "err", err)
		return nil, err
	}

	return targetIPv4, nil
}

func transfer(desc io.WriteCloser, src io.ReadCloser, buffP *sync.Pool) {
	defer desc.Close()
	defer src.Close()

	bf := buffP.Get().(*[]byte)
	_, err := io.CopyBuffer(desc, src, *bf)
	if err != nil {
		return
	}
	buffP.Put(bf)
}

func (ih *InboxHandler) sendHttpRequest(w http.ResponseWriter, r *http.Request) {

	url := fmt.Sprintf("http://%s", r.Host)
	old_body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error clone body request", "err", err)
		http.Error(w, "Server error", http.StatusServiceUnavailable)
		return
	}

	if r.Body != nil {
		r.Body.Close()
	}

	new_body := io.NopCloser(bytes.NewReader(old_body))
	req, err := http.NewRequest(r.Method, url, new_body)
	if err != nil {
		slog.Error("Error clone request", "err", err)
		http.Error(w, "Server error", http.StatusServiceUnavailable)
		return
	}

	req.Header = r.Header.Clone()
	req.Header.Del("Host")

	if len(old_body) > 0 {
		req.ContentLength = int64(len(old_body))
	} 

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		slog.Error("Error send other http req", "err", err, "len(old_body)", len(old_body))
		http.Error(w, "Server error", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, val := range values {
			w.Header().Add(key, val)
		}	
	}

	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func (ih *InboxHandler) workOurHostRequest(w http.ResponseWriter, r *http.Request) {
	
}