package server

import (
	"errors"
	"fmt"
	"sort"

	"github.com/mastercactapus/yaspjs/buffer"
	"github.com/mastercactapus/yaspjs/buffer/grbl"
)

func (srv *Server) defaultBufferTypes() {
	srv.RegisterBufferType("default", buffer.NewDefault)
	srv.RegisterBufferType("grbl", grbl.NewHandler)
}

func (srv *Server) RegisterBufferType(name string, wrap func() buffer.Handler) error {
	if srv.bufferTypeFns[name] != nil {
		return fmt.Errorf("conflicts with existing buffer type '%s'", name)
	}
	if wrap == nil {
		return errors.New("wrap func cannot be nil")
	}

	srv.bufferTypeFns[name] = wrap
	srv.bufferTypeNames = append(srv.bufferTypeNames, name)
	sort.Strings(srv.bufferTypeNames)

	return nil
}
