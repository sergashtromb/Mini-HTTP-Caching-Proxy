package inboxhandler

import (
	"context"
	"net"
	"sync"
)


type ConnManager struct {
	pl 		sync.Pool
	conns 	map[*Session]struct{}
	mu 		sync.Mutex
	wg 		sync.WaitGroup
}

func NewConnManager(buffSize int) *ConnManager {
	return &ConnManager{
		pl: sync.Pool{
			New: func() any {
				buff := make([]byte, buffSize*1024)
				return &buff
			},
		},
		conns: make(map[*Session]struct{}),
	}
}

func (cm *ConnManager) Register(tr net.Conn, cl net.Conn) {

	session := NewSession(tr, cl)
	cm.wg.Add(1)

	cm.mu.Lock()
	cm.conns[session] = struct{}{}
	cm.mu.Unlock()

	session.Start(cm)
}

func (cm *ConnManager) Deregister(se *Session) {

	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.conns, se)
	cm.wg.Done()
}

func (cm *ConnManager) Shutdown(ctx context.Context) error {

	cm.mu.Lock()
	for se, _ := range cm.conns {
		se.Close()
	}
	cm.mu.Unlock()

	// We are waiting for the data transfer to complete.
	done := make(chan struct{})
	go func() {
		cm.wg.Wait()
		close(done)
	}()

	select {
	case <- done:
		return nil
	case <- ctx.Done(): // timeout
		return ctx.Err()
	}
}