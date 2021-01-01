package server

import (
	"fmt"
	"strings"
)

func (srv *Server) handleCommand(data string) {
	srv.send <- data
	parts := strings.SplitN(data, " ", 2)
	cmd := parts[0]
	var argStr string
	if len(parts) > 1 {
		argStr = parts[1]
	}

	switch cmd {
	case "list":
		info, err := srv.ListPorts()
		if err != nil {
			srv.respondErr(fmt.Errorf("list ports: %w", err))
			return
		}
		var res Response
		res.SerialPorts = info
		srv.respondJSON(res)
	case "open":
		srv.handleOpenPort(argStr)
	case "sendjson":
		srv.handleSendJSON(argStr)
	case "send":
		srv.handleSend(argStr)
	// case "sendnobuf":
	// case "bufferalgorithms":
	// case "baudrates":
	case "broadcast":
		srv.send <- argStr
	// case "version":
	// case "hostname":
	default:
		srv.respondErr(fmt.Errorf("unknown command '%s'", cmd))
	}
}
