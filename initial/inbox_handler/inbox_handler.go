// this file is needed to implement the incoming request handler

package inboxhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"fmt"
	"io"
	"log/slog"
	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/domain"
	"mini_http_caching_proxy/tools"
	"net"
	"net/http"
	"slices"

	"golang.org/x/sync/singleflight"
)

type InboxHandler struct {
	client 		http.Client
	cnf 		*config.Config
	buff 		sync.Pool //for copy data in https connect
	cacheStore 	domain.CacheStore
	sgr 		singleflight.Group
	connManager *ConnManager
}


func NewInboxHandler(cnf *config.Config, store domain.CacheStore, cm *ConnManager) *InboxHandler {
	return &InboxHandler{
		client: http.Client{},
		cnf: cnf,
		buff: sync.Pool{
			New: func() interface{} {
				bf := make([]byte, cnf.MemBuff*1024)
				return &bf
			},
		},
		cacheStore: store,
		connManager: cm,
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

func (ih *InboxHandler) Shutdown(ctx context.Context) error {

	if err := ih.connManager.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

type ResultTryTarget struct {
	Res net.Conn
	Err error
}

func createTarget(host string, timeout time.Duration) (net.Conn, error) {

	targetChan := make(chan ResultTryTarget, 2)

	go func() {
		getTarget("tcp4", host, timeout, targetChan)
	}()

	go func() {
		getTarget("tcp6", host, timeout, targetChan)
	}()

	var target net.Conn
	var err []error

	for i := 0; i < 2; i++ {

		res := <- targetChan

		if res.Res != nil {
			target = res.Res
			break
		} else {
			err = append(err, res.Err)
		}
	}

	if target == nil {
		return nil, errors.Join(err...)
	}

	return target, nil
}

func getTarget(tcpV, host string, timeout time.Duration, resChan chan ResultTryTarget) {

	var tarRes ResultTryTarget
	tarRes.Res, tarRes.Err = net.DialTimeout(tcpV, host, timeout)

	select {
	case resChan <- tarRes:
	default:
	}
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
	
	if r.Method == http.MethodGet {

		key := generateKey(r)		
		v, err, _ := ih.sgr.Do(key, func() (any, error) {

			var respAnsw *DataStruct

			data, err := ih.cacheStore.Get(key)
			if err != nil {

				slog.Error("Failed get data from cache", "err", err)

				respFromOurHost, _, err := ih.sendHttpRequestInOurHost(r)
				if err != nil {
					slog.Error("Failed get data from our host", "err", err)
					return nil, err
				}

				respAnsw = respFromOurHost 
				
				return respAnsw, nil
			}
			
			if data == nil {
				
				respFromOurHost, cacheSet, err := ih.sendHttpRequestInOurHost(r)
				if err != nil {
					slog.Error("Failed get data from our host", "err", err)
					return nil, err
				}
				
				sl, err := json.Marshal(respFromOurHost)
				if err != nil {
					slog.Error("indox_handler.go 222: Failed convert DataStruct to []byte", "err", err)
					return nil, err
				}
				slog.Debug("", "cache-control", cacheSet)
				// support cache-control header
				if cacheSet != nil {

					if cacheSet.IsSaved {

						err = ih.cacheStore.SetWithExp(key, sl, cacheSet.LifeTime)
						if err != nil {
							slog.Error("Failed set data in cache store", "err", err)
						}	

					}

				} else {
					err = ih.cacheStore.Set(key, sl)
					if err != nil {
						slog.Error("Failed set data in cache store", "err", err)
					}
				}

				respAnsw = respFromOurHost

			} else {
				
				dt, err := DataStructFromSlice(data)
				if err != nil {
					return nil, err
				}
				respAnsw = dt
			}
			
			return respAnsw, nil
		})

		if err != nil {
			http.Error(w, "Server error", http.StatusServiceUnavailable)
			return
		}

		dt := v.(*DataStruct)
		if dt != nil {

			heads, body := dt.Header, dt.Body

			for key, values := range heads {
				for _, val := range values {
					w.Header().Add(key, val)
				}	
			}

			_, err = w.Write(body)
			if err != nil {
				slog.Error("Failed set body to response", "err", err)
				http.Error(w, "Server error", http.StatusServiceUnavailable)
				return
			}
		}
	} else {
		ih.sendHttpRequest(w, r)
	}
}

func (ih *InboxHandler) sendHttpRequestInOurHost(r *http.Request) (*DataStruct, *CacheSettings, error) {

	url := fmt.Sprintf("http://%s", r.Host)
	old_body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error clone body request", "err", err)
		return nil, nil, err
	}

	if r.Body != nil {
		r.Body.Close()
	}

	new_body := io.NopCloser(bytes.NewReader(old_body))
	req, err := http.NewRequest(r.Method, url, new_body)
	if err != nil {
		slog.Error("Error clone request", "err", err)
		return nil, nil, err
	}

	req.Header = r.Header.Clone()

	if len(old_body) > 0 {
		req.ContentLength = int64(len(old_body))
	} 

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		slog.Error("Error send other http req", "err", err, "len(old_body)", len(old_body))
		return nil, nil, err
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var cacheSet *CacheSettings
	cacheControl := resp.Header.Get("Cache-Control")
	if cacheControl != "" {
		cacheSet = ParceCacheControl(cacheControl)
	}

	ds := &DataStruct {
		Header: resp.Header,
		Body: body,
	}

	return ds, cacheSet, nil
}

func generateKey(r *http.Request) string {

	var strBuilder strings.Builder
	strBuilder.WriteString(r.Method)
	strBuilder.WriteString(r.Host)
	strBuilder.WriteString(r.URL.Path)

	return strBuilder.String()
}

func ParceCacheControl(cacheString string) *CacheSettings {

	arr := strings.Split(cacheString, ",")
	for i, elem := range arr {
		arr[i] = strings.TrimSpace(elem)
	}

	isSaved := true
	var lifeTime int64

	noStore := "no-store"
	maxAge := "max-age"

	for _, elem := range arr {

		if elem == noStore {
			isSaved = false
		} else if strings.Contains(elem, maxAge) {

			_, num, _ := strings.Cut(elem, "=")
			num = strings.TrimSpace(num)

			newNum, err := strconv.ParseInt(num, 10, 64)
			if err != nil {
				lifeTime = 0
			}

			lifeTime = newNum
		}

	}

	return NewCacheSet(isSaved, lifeTime)
}

type CacheSettings struct {
	LifeTime 	time.Duration
	IsSaved 	bool
}

func NewCacheSet(isSaved bool, lifeTime int64) *CacheSettings {
	return &CacheSettings{
		LifeTime: time.Duration(lifeTime) * time.Second,
		IsSaved: isSaved,
	}
}

type DataStruct struct {
	Header 	http.Header `json:"header"`
	Body 	[]byte 		`json:"body"`
}

func DataStructFromSlice(sl []byte) (*DataStruct, error) {

	var dt DataStruct
	err := json.Unmarshal(sl, &dt)
	if err != nil {
		slog.Error("indox_handler.go DataStructFromSlice: Failed convert []byte to DataStruct", "err", err)
		return nil, err
	}

	return &dt, nil
}