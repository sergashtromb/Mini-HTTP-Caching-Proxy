package test

import (
	"encoding/base64"
	inboxhandler "mini_http_caching_proxy/initial/inbox_handler"
	"net/http"
	"testing"
)

func TestParsingProxyAuth(t *testing.T) {
	
	logPas := "user:123"
	logPasBs64 := base64.StdEncoding.EncodeToString([]byte(logPas))

	auth := "Basic " + logPasBs64

	headers := make(http.Header)
	headers.Set("Proxy-Authorization", auth)

	login, pass, ok := inboxhandler.ParseProxyAuthentication(headers)

	if !ok || login != "user" || pass != "123" {
		t.Errorf("Failed parser proxy auth ok=%v log=%v pass=%v", ok, login, pass)
	}
}