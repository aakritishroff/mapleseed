package main

import (
	"testing"
	"fmt"
	"os/exec"
	"os"
	"log"
	"bytes"
	db "./data/inmem"
)

// For now, the top-level test is this: start a server and run
// the crosscloud.js client test suite against it, running under node.js.
//
// That means "go test" requires node.js and npm available on this machine.
func Test_Via_JS(t *testing.T) {
	
	// make port a parameter instead of hardcoding 8090?

	logfile, err := os.OpenFile("Test_Via_JS.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0611);
	if err != nil {
		panic(err)
	}
	log.SetOutput(logfile);

	hubURL := "http://localhost:8090/service"
	cluster := db.NewInMemoryCluster(hubURL)
	cluster.PodURLTemplate = "http://localhost:8090/pod/%s"
	cluster.HubURL = hubURL
	go serve(cluster, ":8090") 
	os.Setenv("PODURL", "http://localhost:8090/pod/testuser/")

	cmd := exec.Command("admin/run-js-tests")
	var stdout,stderr bytes.Buffer
	cmd.Stdout = &stdout	
	cmd.Stderr = &stderr
	err = cmd.Run()
	// maybe only print these if there's an error?
	fmt.Printf("npm test stdout:\n\n%s\n", stdout.String());
	fmt.Printf("npm test stderr:\n\n%s\n", stderr.String());
	if err != nil {
		log.Fatal(err)
	}

	// what brings the http server down, by the way?   I'm surprised
	// this test ever terminates....   but it does.
}


