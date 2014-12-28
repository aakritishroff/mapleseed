package main

import (
	db "./data/inmem"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"testing"
)

// For now, the top-level test is this: start a server and run
// the crosscloud.js client test suite against it, running under node.js.
//
// That means "go test" requires node.js and npm available on this machine.
//
// Don't forget submodules, eg via: go test ./data/...
func Test_Via_JS(t *testing.T) {

	fmt.Printf("bringing up server\n")

	// make port a parameter instead of hardcoding 8090?

	logfile, err := os.OpenFile("Test_Via_JS.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0611)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logfile)

	hubURL := "http://localhost:8090/service"
	cluster := db.NewInMemoryCluster(hubURL)
	cluster.PodURLTemplate = "http://localhost:8090/pod/%s"
	cluster.HubURL = hubURL
	go serve(cluster, ":8090")
	os.Setenv("PODURL", "http://localhost:8090/pod/testuser/")

	fmt.Printf("running crosscloud.js's npm test\n")

	cmd := exec.Command("admin/run-js-tests")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	if err := cmd.Wait(); err != nil {
		panic(err)
	}

	fmt.Printf("bringing down server\n")
	// what brings the http server down, by the way?   I'm surprised
	// this test ever terminates....   but it does.
}
