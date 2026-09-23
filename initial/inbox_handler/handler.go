package inboxhandler

import (
	"context"
	"net/http"
)

type Handler interface {
	InboxRequest(w http.ResponseWriter, r *http.Request)
	Shutdown(ctx context.Context) error
}