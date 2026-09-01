// this file is needed to implement the incoming request handler

package inboxhandler

import (
	"bytes"
	//"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"mini_http_caching_proxy/config"
	"net"
	"net/http"
	"slices"
)

type InboxHandler struct {
	client 	http.Client
	cnf 	*config.Config
}


func NewInboxHandler(cnf *config.Config) *InboxHandler {
	return &InboxHandler{
		client: http.Client{},
		cnf: cnf,
	}
}

func (ih *InboxHandler) HandleInboxReq(w http.ResponseWriter, r *http.Request) {
	
	isOurHost := slices.Contains(ih.cnf.Hosts, r.Host)

	if isOurHost {
		ih.workOurHostRequest(w, r)
	} else {
		ih.workOtherRequest(w, r)
	}

}

func (ih *InboxHandler) workOtherRequest(w http.ResponseWriter, r *http.Request) {

	isHttps := (r.TLS != nil || r.Method == "CONNECT") 
	if isHttps {
		ih.creatTunnel(w, r)
	} else {
		ih.sendHttpRequest(w, r)
	}

}

func (ih *InboxHandler) creatTunnel(w http.ResponseWriter, r *http.Request) {

	// tlsdebug := &tls.Config{
	// 	InsecureSkipVerify: true,
	// 	ServerName: r.Host,
	// }
	slog.Debug("Handler ", "host", r.Host, "method", r.Method)
	target, err := net.Dial("tcp", r.Host)
	if err != nil {
		slog.Error("Error tls.Dial", "err", err)
		http.Error(w, "Server error", http.StatusServiceUnavailable)
		return
	}
	defer target.Close()

	w.WriteHeader(http.StatusOK)
	slog.Debug("tunnel OK")
	hj, ok := w.(http.Hijacker)
	if !ok {
		slog.Error("Hijacker don't supported")
		http.Error(w, "Hijacker don't supported", http.StatusServiceUnavailable)
		return
	}

	clientConn, _, err := hj.Hijack()
	if err != nil {
		slog.Error("Error hijack", "err", err)
		http.Error(w, "Server error", http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	slog.Debug("transfer start")
	go transfer(target, clientConn)
	go transfer(clientConn, target)

}

func transfer(desc io.WriteCloser, src io.ReadCloser) {
	defer desc.Close()
	defer src.Close()
	io.Copy(desc, src)
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

	slog.Debug("req data", "Host", r.Host, "ReqHost", req.Host, "URL", url)

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