package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/mastercactapus/yaspjs/server"
)

var (
	addr = flag.String("addr", ":8989", "HTTP listen address.")
)

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	srv := server.NewServer()

	// TODO: origin
	var upgrader websocket.Upgrader
	upgrader.CheckOrigin = func(req *http.Request) bool { return true }
	http.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
		ws, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Println("ERROR: websocket upgrade:", err)
			return
		}

		defer ws.Close()
		conn := srv.NewConn()
		defer conn.Close()

		cancel := make(chan struct{})
		defer close(cancel)
		go func() {
			defer ws.Close()
			for {
				select {
				case <-conn.Done():
					return
				case <-cancel:
					return
				case msg := <-conn.ToClient():
					err := ws.WriteMessage(websocket.TextMessage, []byte(msg))
					if err != nil {
						log.Println("ERROR: write websocket message:", err)
						return
					}
				}
			}
		}()

		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				log.Println("ERROR: read websocket message:", err)
				return
			}
			select {
			case <-conn.Done():
				return
			case conn.FromClient() <- string(data):
			}
		}
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatalln("ERROR:", err)
	}
}
