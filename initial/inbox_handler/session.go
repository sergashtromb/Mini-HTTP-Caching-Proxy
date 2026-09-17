package inboxhandler

import (
	"io"
	"net"
	"sync"
)

type Session struct {
	target 			net.Conn
	client 			net.Conn
	onseClose 		sync.Once
	onseMajor		sync.Once
}

func NewSession(tr, cl net.Conn) *Session {
	return &Session{
		target: tr,
		client: cl,
	}
}

func (se *Session) Start(cm *ConnManager) {

	sessionClose := func() {
		se.Close()
		cm.Deregister(se)
	}

	go func() {
		transfer(se.target, se.client, &cm.pl)
		se.onseMajor.Do(sessionClose)
	}()

	go func() {
		transfer(se.client, se.target, &cm.pl)
		se.onseMajor.Do(sessionClose)
	}()
}

func (se *Session) Close() {

	se.onseClose.Do(func() {
		if se.target != nil {
			se.target.Close()
		}

		if se.client != nil {
			se.client.Close()
		}
	})
}

func transfer(desc io.WriteCloser, src io.ReadCloser, buffP *sync.Pool) {

	defer desc.Close()
	defer src.Close()	

	bf := buffP.Get().(*[]byte)
	defer buffP.Put(bf)

	_, err := io.CopyBuffer(desc, src, *bf)
	if err != nil {
		return
	}
}