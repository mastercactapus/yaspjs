package server

import (
	"errors"
	"strings"
)

func (srv *Server) handleSend(argStr string) {
	if argStr == "" {
		srv.respondErr(errors.New("missing port"))
		return
	}
	parts := strings.SplitN(argStr, " ", 2)
	if len(parts) == 1 {
		srv.respondErr(errors.New("missing data"))
		return
	}

	ports := <-srv.ports
	p := ports[parts[0]]
	srv.ports <- ports

	if p == nil {
		srv.respondErr(errors.New("specified port not open"))
		return
	}

	srv.respondErr(p.Queue("", parts[1]))
}
