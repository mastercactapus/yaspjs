package server

import "sync/atomic"

type Conn struct {
	id   int32
	srv  *Server
	send chan string
}

func (srv *Server) NewConn() *Conn {
	conn := &Conn{
		id:   atomic.AddInt32(&srv.cid, 1),
		srv:  srv,
		send: make(chan string, 1),
	}
	srv.newConn <- conn

	return conn
}

func (c *Conn) FromClient() chan<- string { return c.srv.input }
func (c *Conn) ToClient() <-chan string   { return c.send }
func (c *Conn) Close()                    { c.srv.closeConn <- c.id }
func (c *Conn) Done() <-chan struct{}     { return c.srv.closeConnsCh }
