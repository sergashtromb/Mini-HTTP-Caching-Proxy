package inboxhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mini_http_caching_proxy/domain"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

type ReversProxy struct {
	sgr 		singleflight.Group
	cacheStore 	domain.CacheStore
}

func NewReveresProxy(cs domain.CacheStore) *ReversProxy {
	return &ReversProxy{
		cacheStore: cs,
	}
}

func (rp *ReversProxy) InboxRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {

		key := generateKey(r)		
		v, err, _ := rp.sgr.Do(key, func() (any, error) {

			var respAnsw *DataStruct

			data, err := rp.cacheStore.Get(key)
			if err != nil {

				slog.Error("Failed get data from cache", "err", err)

				respFromOurHost, _, err := rp.sendHttpRequestInOurHost(r)
				if err != nil {
					slog.Error("Failed get data from our host", "err", err)
					return nil, err
				}

				respAnsw = respFromOurHost 
				
				return respAnsw, nil
			}
			
			if data == nil {
				
				respFromOurHost, cacheSet, err := rp.sendHttpRequestInOurHost(r)
				if err != nil {
					slog.Error("Failed get data from our host", "err", err)
					return nil, err
				}
				
				sl, err := json.Marshal(respFromOurHost)
				if err != nil {
					slog.Error("indox_handler.go 222: Failed convert DataStruct to []byte", "err", err)
					return nil, err
				}
				
				// support cache-control header
				if cacheSet != nil {

					if cacheSet.IsSaved {

						err = rp.cacheStore.SetWithExp(key, sl, cacheSet.LifeTime)
						if err != nil {
							slog.Error("Failed set data in cache store", "err", err)
						}	

					}

				} else {
					err = rp.cacheStore.Set(key, sl)
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
		sendHttpRequest(w, r)
	}
}

func (rp *ReversProxy) sendHttpRequestInOurHost(r *http.Request) (*DataStruct, *CacheSettings, error) {

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
	defer resp.Body.Close()

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

func (rp *ReversProxy) Shutdown(ctx context.Context) error {
	return nil
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