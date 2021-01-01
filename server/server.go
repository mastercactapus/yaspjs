package server

import (
	"github.com/mastercactapus/yaspjs/buffer"
)

type Server struct {
	cid int32

	conns chan []*Conn

	ports chan map[string]*Port

	bufferTypeNames []string
	bufferTypeFns   map[string]func() buffer.Handler

	input chan string
	send  chan string

	newConn   chan *Conn
	closeConn chan int32

	closeConnsCh chan struct{}
}

func NewServer() *Server {
	srv := &Server{
		input:         make(chan string),
		newConn:       make(chan *Conn),
		closeConn:     make(chan int32),
		closeConnsCh:  make(chan struct{}),
		send:          make(chan string, 1),
		conns:         make(chan []*Conn, 1),
		ports:         make(chan map[string]*Port, 1),
		bufferTypeFns: make(map[string]func() buffer.Handler),
	}
	srv.conns <- nil
	srv.ports <- make(map[string]*Port)
	srv.defaultBufferTypes()

	go srv.loop()
	go srv.sendLoop()
	return srv
}

func (srv *Server) sendLoop() {
	for data := range srv.send {
		conns := <-srv.conns
		srv.conns <- conns

		for _, c := range conns {
			c.send <- data
		}
	}
}

func (srv *Server) loop() {
	for {
		select {
		case command := <-srv.input:
			srv.handleCommand(command)
		case c := <-srv.newConn:
			conns := <-srv.conns
			srv.conns <- append(conns, c)
		case id := <-srv.closeConn:
			origConns := <-srv.conns
			conns := origConns[:0]
			for _, c := range origConns {
				if c.id == id {
					continue
				}
				conns = append(conns, c)
			}
			srv.conns <- conns
		}
	}
}
