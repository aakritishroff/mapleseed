package main

import (
	"log"
	"code.google.com/p/go.net/websocket"
	"net/http"
	"github.com/sandhawke/inmem/db"
)

type JSON map[string]interface{};

var cluster *db.Cluster

func serve(name, portString string) {
	cluster = db.NewInMemoryCluster("http://example.com")
	log.Printf("Answering on http://localhost%s/_ws", portString)
	http.Handle("/_ws", websocket.Handler(webHandler))
	err := http.ListenAndServe(portString, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func main() {
	serve("http://example.com", ":8087")
}

