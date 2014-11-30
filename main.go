package main

import (
	"log"
	"code.google.com/p/go.net/websocket"
	"net/http"
	"github.com/sandhawke/inmem/db"
	"flag"
	"os"
	"time"
	"fmt"
)

type JSON map[string]interface{};

var cluster *db.Cluster

var podURLTemplate string

func serve(hubURL, portString string) {
	cluster = db.NewInMemoryCluster(hubURL)
	log.Printf("Answering on %s%s", hubURL, portString)
	http.Handle("/.well-known/podsocket/v1", websocket.Handler(websocketHandler))
    http.HandleFunc("/", httpHandler)
	err := http.ListenAndServe(portString, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func main() {

	var hubURL = flag.String("hubURL", "http://localhost", "main URL of service")
	var argPodURLTemplate = flag.String("pods", "http://localhost:8080/pod/%s", "URLs of created pods, with %s as the pod name")
	var port = flag.String("port", "8080", "web server port")
	var logdir = flag.String("logdir", "/var/log/mapleseed", "where to put the log files")
	var dolog = flag.Bool("log", false, "log to file instead of stdout")
	// var restore = flag.String("restore", "", "restore state from given json dump file")
	flag.Parse()

	podURLTemplate = *argPodURLTemplate

	if *dolog {
		err := os.MkdirAll(*logdir, 0700)
		if err != nil {
			panic(err)
		}
		logfilename := *logdir+"/log-"+time.Now().Format("20060102")
		fmt.Printf("logging to %s\n", logfilename)
		logfile, err := os.OpenFile(logfilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0611);
		if err != nil {
			// ??? not sure why I'm getting "No such file or directory" 
			panic(err)
		}
		log.SetOutput(logfile)
	}

	/*
	if *restore != "" {
		fi, err := os.Open(*restore)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
		restoreCluster(fi)
	}
    */

	serve(*hubURL, ":"+*port)
}

