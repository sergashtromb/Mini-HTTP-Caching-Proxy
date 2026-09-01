package tools

import (
	"crypto/tls"
	"log/slog"
	"net/http"
)

func DefineDEBUGTransport() *http.Transport {
	slog.Warn("DON'T CHECK TSL", "InsecureSkipVerify", true)
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
}