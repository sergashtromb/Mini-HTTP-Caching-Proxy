package inboxhandler

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

func sendHttpRequest(w http.ResponseWriter, r *http.Request) {

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