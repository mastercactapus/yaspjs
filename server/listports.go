package server

import (
	"errors"
	"sort"
)

type SerialPortInfo struct {
	Name         string
	FriendlyName string `json:"Friendly"`
	SerialNumber string
	DeviceClass  string
	ProductID    string  `json:"UsbPid"`
	VendorID     string  `json:"UsbVid"`
	Version      float32 `json:"Ver"`
	RelatedNames []string

	IsOpen          bool
	IsPrimary       bool
	Baud            int
	BufferAlgorithm string

	AvailableBufferAlgorithms []string
}

var nativeListPorts func() ([]SerialPortInfo, error)

func (srv *Server) ListPorts() ([]SerialPortInfo, error) {
	if nativeListPorts == nil {
		return nil, errors.New("unsupported on this platform")
	}

	info, err := nativeListPorts()
	if err != nil {
		return nil, err
	}
	sort.Slice(info, func(i, j int) bool { return info[i].Name < info[j].Name })

	ports := <-srv.ports
	for i := range info {
		info[i].AvailableBufferAlgorithms = srv.bufferTypeNames
		p := ports[info[i].Name]
		if p == nil {
			continue
		}

		info[i].IsOpen = true
		info[i].IsPrimary = p.primary
		info[i].Baud = p.baudRate
		info[i].BufferAlgorithm = p.bufferType
	}
	srv.ports <- ports

	return info, nil
}
