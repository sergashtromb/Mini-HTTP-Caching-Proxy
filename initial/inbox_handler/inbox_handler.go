// this file is needed to implement the incoming request handler

package inboxhandler

import (
	"net/http"
	"mini_http_caching_proxy/config"
)

type InboxHandler struct {
	client 	http.Client
	cnf 	*config.Config
}


func NewInboxHandler() *InboxHandler {
	return &InboxHandler{
		client: http.Client{},
	}
}

func (ih *InboxHandler) HandleInboxReq(w http.ResponseWriter, r *http.Request) {



}