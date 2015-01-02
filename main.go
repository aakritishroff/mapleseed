package main

import (
	"log"
	"net/http"
	"flag"
	"os"
	"time"
	"fmt"
	db "github.com/sandhawke/mapleseed/data/inmem"
	httpTransport "github.com/sandhawke/mapleseed/transport/http"
	wsTransport "github.com/sandhawke/mapleseed/transport/websocket"
)

type JSON map[string]interface{};

func serve(cluster *db.Cluster, portString string) {
	log.Printf("Answering on %s%s", cluster.HubURL, portString)
	
	wsTransport.Register(cluster);

	hh := func(w http.ResponseWriter, r *http.Request) {
		httpTransport.Handler(cluster, w, r)
	}
    http.HandleFunc("/", hh)
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
	var restore = flag.String("restore", "", "restore state from given json dump file")
	flag.Parse()

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

	cluster := db.NewInMemoryCluster()
	cluster.PodURLTemplate = *argPodURLTemplate
	cluster.HubURL = *hubURL

	if *restore != "" {
		fi, err := os.Open(*restore)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
		err = cluster.RestoreFrom(fi)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
	}

	serve(cluster, ":"+*port)
}

