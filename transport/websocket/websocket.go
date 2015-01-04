/*

Provide access via a WebSocket.

Mirrors ./http.go in functionality.

Constucts an appropriate Act and calls the selected op function
to handle the client request.

*/

package websocket

import (
	"log"
	// "fmt"
	"code.google.com/p/go.net/websocket"
	"io"
	"net/http"
	// "net/http"
	// "net"
	db "github.com/sandhawke/mapleseed/data/inmem"
	"github.com/sandhawke/mapleseed/op"
)


var Trace bool
func init() {
	Trace = true
}

func trace(template string, args ...interface{}) {
	if Trace {
		log.Printf("ws."+template, args...)
	}
}

type InMessage struct {
	Seq  int     `json:"seq"`
	Op   string  `json:"op"`
	Data op.JSON `json:"data"`
}

type OutMessage struct {
	InReplyTo int     `json:"inReplyTo"`
	Final     bool    `json:"final"`
	Op        string  `json:"op"`
	Data      op.JSON `json:"data"`
}

type WSAct struct {
	ws      *websocket.Conn
	seq     int
	pod     *db.Pod
	cluster *db.Cluster
	userId  string
	closed  bool
}

func (act *WSAct) Event(op string, data op.JSON) {
	act.sendRaw(OutMessage{act.seq, false, op, data})
}

func (act *WSAct) Result(data op.JSON) {
	act.sendRaw(OutMessage{act.seq, true, "ok", data})
	act.closed = true
}

// we need to formalize this more at some point.   Maybe
// flags for types of errors?
func (act *WSAct) Error(code int16, message string, details op.JSON) {
	act.sendRaw(OutMessage{act.seq, true, "err",
		op.JSON{"text": message}})
	act.closed = true
}

func (act *WSAct) Closed() bool {
	return act.closed
}

func (act *WSAct) Pod() *db.Pod {
	return act.pod
}

func (act *WSAct) Cluster() *db.Cluster {
	return act.cluster
}

func (act *WSAct) UserId() string {
	return act.userId
}

func (act *WSAct) sendRaw(msg OutMessage) {
<<<<<<< HEAD
	log.Printf("--> %q", msg)
	if act.closed {
		panic("who is trying to send when act.closed?")
=======
	trace("sendRaw %+v", msg)	
	if act.closed { 
		return
		//  -- at some point turn this back on -- 
		// panic("who is trying to send when act.closed?") 
>>>>>>> 638a14e81690878c70f746c7e42824bc82a81bb9
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

func Register(cluster *db.Cluster) {
	ws := func(ws *websocket.Conn) {
		handler(cluster, ws)
	}
	http.Handle("/.well-known/podsocket/v1", websocket.Handler(ws))
}

func handler(cluster *db.Cluster, ws *websocket.Conn) {

	origin := ws.LocalAddr() // then turn into domain name?

	// @@@  defer:  stop any queries we've started

	nextSeq := 0
	userId := ""
	var userPod *db.Pod
	var thisPod *db.Pod

	// figure out thisPod by looking at the URL


	trace("------------- new websocket -------------")
	trace("handler localAddr   %+v", ws.LocalAddr)
	trace("handler remoteAddr  %+v", ws.RemoteAddr)

	for {
		in := InMessage{nextSeq, "nop", nil}
		if err := websocket.JSON.Receive(ws, &in); err != nil {
			if err == io.EOF {
				return
			}
			trace("recv from %s err in receive: %q\n", origin, err)
			return
		}
		nextSeq = in.Seq + 1
		trace("--recv--")
		trace("recv'd: %+v", in)

		act := &WSAct{ws, in.Seq, thisPod, cluster, userId, false}

		/*
			var url string
			switch in.Op {
			case "read", "overlay", "delete", "stopQuery":
				url = in.Data["_id"].(string)

				if (!act.inMySpace(url)) {
					act.Send(Message{in.Seq, "fail",
						op.JSON{"err":"requested URL not on this pod"}})
					return
				}
			}
		*/

		switch in.Op {
		case "login":

			userId, _ = in.Data["userId"].(string)
			password, _ := in.Data["password"].(string)

			userPod = cluster.PodByURL(userId)
			if userPod == nil {
				// if pw is nil, then assume this is a <0.2.0 client
				// and for now allow implicit creatione
				if password == "" {
					userPod = db.NewPod(userId)
					err := cluster.AddPod(userPod)
					// we know the name wasn't taken, per above
					if err != nil {
						panic(err)
					}
				} else {
					act.Error(404, "No such user", op.JSON{})
					userId = ""
				}
			} else {
				log.Printf("100 %q", userPod)
				if userPod.HasPassword(password) {
					trace("login worked for %s", userId)
				} else {
					act.Error(401, "Incorrect Password", op.JSON{})
					userPod = nil
					userId = ""
				}
			}
			trace("userPod=%+v", userPod)
			trace("userId=%v", userId)
			if userPod != nil {
				act.Result(nil)
			}

		case "whoami":
			trace("still logged in %s", userId)
			act.Result(op.JSON{"userId":userId})


		case "createPod":
			name, _ := in.Data["name"].(string)
			password, _ := in.Data["password"].(string)
			op.CreatePod(act, name, password)

		case "create":
			options := op.CreationOptions{}
			trace("op=create options=%q",in.Data)
			options.InContainer,_ = in.Data["inContainer"].(string)
			options.SuggestedName,_ = in.Data["suggestedName"].(string)
			options.RequiredId,_ = in.Data["requiredId"].(string)


			// I don't quite understand why we can't call it op.JSON here, but
			// when we do, the value gets silently lost
			options.InitialData, _ = in.Data["initialData"].(map[string]interface{})
			options.IsConstant, _ = in.Data["isConstant"].(bool)
			op.Create(act, options)

		case "read":
			url, _ := in.Data["_id"].(string)
			op.Read(act, url)

		case "update":
			url, _ := in.Data["_id"].(string)
			onlyIfMatch, _ := in.Data["_etag"].(string)
			op.Update(act, url, onlyIfMatch, in.Data)

		case "delete":
			url := in.Data["_id"].(string)

			op.Delete(act, url)

		case "startQuery":
			options := op.QueryOptions{}
			options.InContainer, _ = in.Data["inContainer"].(string)
			limit, limitGiven := in.Data["limit"].(float64)
			if limitGiven {
				options.Limit = uint32(limit)
			}
			options.Filter, _ = in.Data["filter"].(map[string]interface{})
			events, eventsGiven := in.Data["events"].(map[string]interface{})
			if eventsGiven {
				options.Watching_AllResults, _ = events["AllResults"].(bool)
				options.Watching_Progress, _ = events["Progress"].(bool)
				options.Watching_Appear, _ = events["Appear"].(bool)
				options.Watching_Disappear, _ = events["Disappear"].(bool)
			} else {
				options.Watching_AllResults = true
				options.Watching_Progress = true
				options.Watching_Appear = true
				options.Watching_Disappear = true
			}

			trace("op=startQuery options=%q, parsed=%q", in.Data, options)

			op.StartQuery(act, options)

		case "stopQuery":
			id, _ := in.Data["_id"].(string)
			op.StopQuery(act, id)

		case "ping":
			act.Result(op.JSON{"isPong": true, "modCount": cluster.ModCount()})

		default:
			trace("Unimplemented op: %s", in.Op)
			act.Error(400, "Operation unknown or unimplemented", op.JSON{})

		}
	}
}
