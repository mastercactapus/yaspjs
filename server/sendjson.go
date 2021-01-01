package server

import (
	"encoding/json"
	"errors"
)

func (srv *Server) handleSendJSON(argStr string) {
	// can use same format
	var req Response
	err := json.Unmarshal([]byte(argStr), &req)
	if err != nil {
		srv.respondErr(err)
		return
	}

	ports := <-srv.ports
	p := ports[req.P]
	srv.ports <- ports

	if p == nil {
		srv.respondErr(errors.New("specified port not open"))
		return
	}

	for _, data := range req.Data {
		err := p.Queue(data.ID, data.D)
		if err != nil {
			srv.respondErr(err)
			return
		}
	}
}
