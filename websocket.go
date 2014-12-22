/*

Provide access via a WebSocket.

Mirrors ./http.go in functionality.

Constucts an appropriate Act and calls the selected op function
to handle the client request.

*/

package main

import (
	"code.google.com/p/go.net/websocket"
	//"fmt"
	"io"
	"log"
	// "net/http"
	// "net"
	db "github.com/aakritishroff/datapages/inmem"
	//db "github.com/sandhawke/pagestore/inmem"
)

type InMessage struct {
	Seq  int    `json:"seq"`
	Op   string `json:"op"`
	Data JSON   `json:"data"`
}

type OutMessage struct {
	InReplyTo int    `json:"inReplyTo"`
	Final     bool   `json:"final"`
	Op        string `json:"op"`
	Data      JSON   `json:"data"`
}

type WSAct struct {
	ws     *websocket.Conn
	seq    int
	pod    *db.Pod
	userId string
	closed bool
}

func (act *WSAct) Event(op string, data JSON) {
	act.sendRaw(OutMessage{act.seq, false, op, data})
}

func (act *WSAct) Result(data JSON) {
	act.sendRaw(OutMessage{act.seq, true, "ok", data})
	act.closed = true
}

// we need to formalize this more at some point.   Maybe
// flags for types of errors?
func (act *WSAct) Error(code int16, message string, details JSON) {
	act.sendRaw(OutMessage{act.seq, true, "err",
		JSON{"text": message}})
	act.closed = true
}

func (act *WSAct) Closed() bool {
	return act.closed
}

func (act *WSAct) Pod() *db.Pod {
	return act.pod
}

func (act *WSAct) UserId() string {
	return act.userId
}

func (act *WSAct) sendRaw(msg OutMessage) {
	log.Printf("--> %q", msg)
	if act.closed {
		panic("who is trying to send when act.closed?")
	}
	err := websocket.JSON.Send(act.ws, msg)
	if err != nil {
		log.Printf("websocket from XX err in send: %q\n", err)
		log.Printf("act.closed=%b", act.closed)
		act.closed = true
		log.Printf("act.closed=%b", act.closed)
		// actually, mark EVERY act on this websocket closed, not just
		// this one!
	}
}

func websocketHandler(ws *websocket.Conn) {
	origin := ws.LocalAddr() // then turn into domain name?

	// @@@  defer:  stop any queries we've started

	nextSeq := 0
	userId := ""
	var pod *db.Pod

	for {
		in := InMessage{nextSeq, "nop", nil}
		if err := websocket.JSON.Receive(ws, &in); err != nil {
			if err == io.EOF {
				return
			}
			log.Printf("websocket from %s err in receive: %q\n", origin, err)
			return
		}
		nextSeq = in.Seq + 1
		log.Printf("Received: %q\n", in)

		act := &WSAct{ws, in.Seq, pod, userId, false}

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

			// Later on, we'll require a token obtained via a direct channel

			// for now, we basically treat the userId (user pod url) as
			// an opaque string!   (I think...)

			userId = in.Data["userId"].(string)
			pod, _ = cluster.NewPod(userId)
			log.Printf("logged in %s", userId)
			log.Printf("pod URL is %s", pod.URL())
			act.Result(nil)

		case "whoami":
			log.Printf("still logged in %s", userId)
			act.Result(JSON{"userId": userId})

		case "createPod":
			name, _ := in.Data["name"].(string)
			createPod(act, name)

		case "create":
			options := CreationOptions{}
			log.Printf("op=create options=%q", in.Data)
			options.inContainer, _ = in.Data["inContainer"].(string)
			options.suggestedName, _ = in.Data["suggestedName"].(string)
			options.requiredId, _ = in.Data["requiredId"].(string)

			// I don't quite understand why we can't call it JSON here, but
			// when we do, the value gets silently lost
			options.initialData, _ = in.Data["initialData"].(map[string]interface{})
			options.isConstant, _ = in.Data["isConstant"].(bool)
			create(act, options)

		case "read":
			url, _ := in.Data["_id"].(string)
			read(act, url)

		case "update":
			url, _ := in.Data["_id"].(string)
			onlyIfMatch, _ := in.Data["_etag"].(string)
			update(act, url, onlyIfMatch, in.Data)

		case "delete":
			url := in.Data["_id"].(string)

			pageDelete(act, url)

		case "startQuery":
			options := QueryOptions{}
			options.inContainer, _ = in.Data["inContainer"].(string)
			limit, limitGiven := in.Data["limit"].(float64)
			if limitGiven {
				options.limit = uint32(limit)
			}
			options.filter, _ = in.Data["filter"].(map[string]interface{})
			events, eventsGiven := in.Data["events"].(map[string]interface{})
			if eventsGiven {
				options.watching_AllResults, _ = events["AllResults"].(bool)
				options.watching_Progress, _ = events["Progress"].(bool)
				options.watching_Appear, _ = events["Appear"].(bool)
				options.watching_Disappear, _ = events["Disappear"].(bool)
			} else {
				options.watching_AllResults = true
				options.watching_Progress = true
				options.watching_Appear = true
				options.watching_Disappear = true
			}

			log.Printf("op=startQuery options=%q, parsed=%q", in.Data, options)

			startQuery(act, options)

		case "stopQuery":
			id, _ := in.Data["_id"].(string)
			stopQuery(act, id)

		case "ping":
			act.Result(JSON{"isPong": true, "modCount": cluster.ModCount()})

		default:
			log.Printf("Unimplemented op: %s", in.Op)
			act.Error(400, "Operation unknown or unimplemented", JSON{})

		}
	}
}
