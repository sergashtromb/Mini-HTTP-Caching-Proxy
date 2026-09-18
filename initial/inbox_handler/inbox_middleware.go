package inboxhandler

import (
	"encoding/base64"
	"log/slog"
	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/domain"
	"mini_http_caching_proxy/rate"
	"net"
	"net/http"
	"slices"
	"strings"
)

type Middleware struct {
	cnf 			*config.Config
	generalLimiter 	*rate.Limiter
	shardLimiter 	*rate.ShardLimiter
}

func NewMiddleware(cnf *config.Config, gl *rate.Limiter, sh *rate.ShardLimiter) *Middleware {
	return &Middleware {
		cnf: cnf,
		generalLimiter: gl,
		shardLimiter: sh,
	} 
}

func (mi *Middleware) InternalHostMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		
		inboxRequest, err := createInboxReq(r)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return 
		}

		var checkRL bool
		intHost := slices.Contains(mi.cnf.Hosts, inboxRequest.Host)
		
		if intHost {
			checkRL = mi.generalLimiter.Allow()

		} else {
			checkRL = mi.shardLimiter.Allow(inboxRequest.IP)
		}

		if !checkRL {
			http.Error(w, "Too many request", http.StatusTooManyRequests)
			return 
		}
		
		next.ServeHTTP(w, r)
	})
}

func (mi *Middleware) CheckAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		login, pass, ok := "", "", false

		if r.Method == http.MethodConnect {
			login, pass, ok = ParseProxyAuthentication(r.Header)
		} else {
			login, pass, ok = r.BasicAuth()
		}

		if !checkUser(login, pass) || !ok {
			w.Header().Set("Proxy-Authenticate", `Basic realm="mini-proxy"`)
			w.Header().Set("Content-Length", "0")
			w.WriteHeader(http.StatusProxyAuthRequired)
			return 
		}

		next.ServeHTTP(w, r)
	})
}

// A stub to simulate the operation of a database check
func checkUser(user, password string) bool {
	
	if user == "" || password == ""{
		return false
	}

	slog.Debug("user checked", "user", user, "pass", password)
	return true
}

func ParseProxyAuthentication(headers http.Header) (string, string, bool) {

	auth := headers.Get("Proxy-Authorization")
	if auth == "" {
		return "", "", false
	}

	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return "", "", false
	}

	code := auth[len(prefix):]
	decodBt, err := base64.StdEncoding.DecodeString(code)
	if err != nil {
		return "", "", false
	}

	login, pass, ok := strings.Cut(string(decodBt), ":")
	if !ok {
		return "", "", false
	}

	return strings.TrimSpace(login), strings.TrimSpace(pass), true
}

func createInboxReq(r *http.Request) (*domain.InboxRequest,  error) {

	ip, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		slog.Error("Error get ip and port from req", "err", err)
		return nil, err
	}

	inboxReq := domain.InboxRequest {
		Method: r.Method,
		IP: ip,
		Port: port,
		Host: r.Host,
		Path: r.URL.Path,
	}

	return &inboxReq, nil
}