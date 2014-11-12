/*

Provide access via a WebSocket.    

Mirrors ./http.go in functionality.

Constucts an appropriate Act and calls the selected op function
to handle the client request.

*/

package main

import (
	"log"
	"fmt"
	"io"
	"code.google.com/p/go.net/websocket"
	// "net/http"
	// "net"
	// "github.com/sandhawke/inmem/db"
)


type InMessage struct {
	Seq int `json:"seq"`
	Op string `json:"op"`
	Data JSON `json:"data"`
}

type OutMessage struct {
	InReplyTo int `json:"inReplyTo"`
	Final bool `json:"final"`
	Op string `json:"op"`
	Data JSON `json:"data"`
}

type WSAct struct {
	ws *websocket.Conn
	seq int
}

func (act WSAct) Send(op string, data JSON) {
	act.sendRaw(OutMessage{act.seq, false, op, data})
}

func (act WSAct) SendFinal(op string, data JSON) {
	act.sendRaw(OutMessage{act.seq, true, op, data})
}

func (act WSAct) Error(code uint32, message string) {
	act.sendRaw(OutMessage{act.seq, true, "err", 
		JSON{"text": message}})
}

func (act WSAct) sendRaw(msg OutMessage) {
	err := websocket.JSON.Send(act.ws, msg)
	if err != nil {
		log.Printf("websocket from XX err in send: %q\n", err)
		//act.Close() or something like that?
	}
}

func webHandler(ws *websocket.Conn) {

	origin := ws.LocalAddr() // then turn into domain name?

	// @@@  defer:  stop any queries we've started

	nextSeq := 0

	for {
		in := InMessage{nextSeq, "nop", nil}
		if err := websocket.JSON.Receive(ws, &in); err != nil {
			if err == io.EOF {
				return; 
			}
			log.Printf("websocket from %s err in receive: %q\n", origin, err)
			return
		}
		nextSeq = in.Seq + 1
		fmt.Printf("Received: %q\n", in)

		act := WSAct{ws,in.Seq}

		/*
		var url string
		switch in.Op {
		case "read", "overlay", "delete", "stopQuery":
			url = in.Data["_id"].(string)

			if (!act.inMySpace(url)) {
				act.Send(Message{in.Seq, "fail", 
					JSON{"err":"requested URL not on this pod"}})
				return
			}
		}
        */

		switch in.Op {
		case "login":
			// For now we don't do authentication...
			//
			// Later on, we'll require a token obtained via a direct channel
			// userId := in.Data["userId"].(string)
			//  pod,_ = cluster.NewPod(act.userId)
			act.SendFinal("ok", nil)

		case "create":

			options := CreationOptions{}
			options.inContainer = in.Data["inContainer"].(*string)
			options.suggestedName = in.Data["suggestedName"].(*string)
			options.requiredId = in.Data["requiredId"].(*string)
			options.initialData = in.Data["initialData"].(*JSON)
			c := in.Data["isConstant"].(*bool)
			options.isConstant = ( c != nil && *c )

			create(act, options)

		case "read":
			url := in.Data["_id"].(string)
			read(act, url)
			
		case "overlay":
			url := in.Data["_id"].(string)
			onlyIfMatch := ""
			if x := in.Data["_etag"]; x != nil {
				onlyIfMatch = x.(string)
			} 
			overlay(act, url, onlyIfMatch, in.Data)

		case "delete":
			url := in.Data["_id"].(string)

			pageDelete(act, url)

			/*
		case "startQuery":
			startQuery(act, in)
		case "stopQuery":
			stopQuery(act, in)
*/
		}
	}
}
