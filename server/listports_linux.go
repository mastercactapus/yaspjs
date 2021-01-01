package server

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func init() {
	nativeListPorts = linuxListPorts
}

func linuxListPorts() ([]SerialPortInfo, error) {
	var devicePaths []string
	err := filepath.Walk("/sys/devices", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if !info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, "ttyA") || strings.HasPrefix(base, "ttyU") {
			devicePaths = append(devicePaths, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	info := make([]SerialPortInfo, 0, len(devicePaths))
	for _, path := range devicePaths {
		portName := filepath.Base(path)
		for path != "/" {
			path = filepath.Dir(path)

			data := func(name string) []byte {
				data, _ := ioutil.ReadFile(filepath.Join(path, name))
				return data
			}
			str := func(name string) string { return strings.TrimSpace(string(data(name))) }
			float := func(name string) float32 {
				f, _ := strconv.ParseFloat(str(name), 32)
				return float32(f)
			}

			_, err := os.Stat(filepath.Join(path, "product"))
			if os.IsNotExist(err) {
				_, err = os.Stat(filepath.Join(path, "manufacturer"))
			}
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, err
			}

			var related []string
			for _, relPath := range devicePaths {
				if !strings.HasPrefix(relPath, path) {
					continue
				}
				base := filepath.Base(relPath)
				if base == portName {
					continue
				}
				related = append(related, "/dev/"+base)
			}

			info = append(info, SerialPortInfo{
				Name:         "/dev/" + portName,
				RelatedNames: related,
				DeviceClass:  str("bDeviceClass"),
				FriendlyName: str("product"),
				VendorID:     str("idVendor"),
				ProductID:    str("idProduct"),
				Version:      float("version"),
				SerialNumber: str("serial"),
			})
			break
		}
	}

	return info, nil
}
