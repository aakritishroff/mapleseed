/*

Provide access via HTTP, along the lines of LDP 1.0

Mirrors ./websocket.go in functionality.

Constucts an appropriate Session and calls the appropite op functions
based on client requests.



     THIS CODE IS BORROWED FROM FAKEPODS.GO AND NOT FULLY UPDATED


*/

package main

import ( 
	"log"
	"net/http"
)

type HTTPSession struct {

}

func httpHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\n");
	log.Printf("Request %q\n", r)

}
