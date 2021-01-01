package server

import (
	"encoding/json"
	"log"
)

type Response struct {
	SerialPorts []SerialPortInfo `json:",omitempty"`
	Cmd         string           `json:",omitempty"`
	Desc        string           `json:",omitempty"`
	Port        string           `json:",omitempty"`

	Baud       int    `json:",omitempty"`
	BufferType string `json:",omitempty"`
	IsPrimary  bool   `json:",omitempty"`

	QCnt int

	Data []struct {
		D  string
		ID string `json:"Id"`
	} `json:",omitempty"`

	ID        string `json:"Id,omitempty"`
	P, D      string `json:",omitempty"`
	ErrorCode string `json:",omitempty"`
}

func (srv *Server) respondJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		panic(err)
	}

	srv.send <- string(data)
}
func (srv *Server) respondErr(err error) {
	if err == nil {
		return
	}
	log.Println("ERROR:", err)
	var data struct {
		Error string
	}
	data.Error = err.Error()

	srv.respondJSON(data)
}
