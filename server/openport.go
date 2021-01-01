package server

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/mastercactapus/yaspjs/buffer"
	"go.bug.st/serial"
)

func (srv *Server) handleOpenPort(argStr string) {
	var err error
	args := strings.Fields(argStr)
	var res Response
	switch len(args) {
	case 0:
		res.Cmd = "OpenFail"
		res.Desc = "missing port name"
	case 1:
		res.Cmd = "OpenFail"
		res.Desc = "missing baud rate"
	case 2:
		args = append(args, "default")
		fallthrough
	case 3:
		res.Cmd = "Open"
		res.Desc = "Got register/open on port."
		res.Port = args[0]
		res.Baud, err = strconv.Atoi(args[1])
		if err != nil {
			res.Cmd = "OpenFail"
			res.Desc = fmt.Sprintf("invalid baud rate: %v", err)
			break
		}
		res.IsPrimary, err = srv.OpenPort(res.Port, res.Baud, args[2])
		if err != nil {
			res.Cmd = "OpenFail"
			res.Desc = err.Error()
		}
	}
	srv.respondJSON(res)
}

func (srv *Server) OpenPort(name string, baud int, bufferType string) (bool, error) {
	if baud == 0 {
		return false, errors.New("missing baud rate")
	}
	newBuf := srv.bufferTypeFns[bufferType]
	if newBuf == nil {
		return false, fmt.Errorf("unknown/unsupported buffer type '%s'", bufferType)
	}

	ports := <-srv.ports
	p := ports[name]
	if p != nil {
		srv.ports <- ports

		// already open
		return p.primary, nil
	}

	sp, err := serial.Open(name, &serial.Mode{BaudRate: baud})
	if err != nil {
		srv.ports <- ports
		return false, fmt.Errorf("open port: %w", err)
	}
	primary := len(ports) == 0
	p = &Port{
		name:       name,
		baudRate:   baud,
		bufferType: bufferType,
		primary:    primary,
		Buffer: buffer.NewBuffer(buffer.Config{
			PollInterval:    3 * time.Second,
			ReadWriteCloser: sp,
			Handler:         newBuf(),
			OnRead: func(line string) {
				srv.respondJSON(Response{
					P: name,
					D: line,
				})
			},
			OnUpdate: func(cmd buffer.CommandResponse) {
				if cmd.ID == "" {
					return
				}
				res := Response{
					P:    name,
					QCnt: p.WriteQueueLen(),
					D:    cmd.Data,
					ID:   cmd.ID,
				}
				switch {
				case cmd.Queued:
					res.Cmd = "Queued"
				case cmd.Sent:
					res.Cmd = "Write"
				case cmd.Done:
					res.Cmd = "Complete"
				case cmd.Err != nil:
				default:
					log.Printf("unknown update from %s: %v", bufferType, cmd)
					return
				}

				if cmd.Seq > 1 {
					cmd.ID = fmt.Sprintf("%s-part-%d-%d", cmd.ID, cmd.Seq, cmd.SeqMax)
				}

				srv.respondJSON(res)
			},
		}),
	}
	ports[name] = p
	srv.ports <- ports

	return primary, nil
}
