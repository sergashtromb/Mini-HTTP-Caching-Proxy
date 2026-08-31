package inboxhandler

import (
	"log/slog"
	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/domain"
	"mini_http_caching_proxy/rate"
	"net"
	"net/http"
	"slices"
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
		Host: r.URL.Host,
		Path: r.URL.Path,
	}

	return &inboxReq, nil
}