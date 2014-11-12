package main

import (
	"log"
	"fmt"
	"io"
	"code.google.com/p/go.net/websocket"
	"net/http"
	"net"
	"github.com/sandhawke/inmem"
)

type JSON map[string]interface{};

type Message struct {
	Seq int
	Op string
	Data JSON
}

type Session struct {

	closed bool

	// authenticated user
	userId string
	pod *inmem.Pod

	// local ids

	// binary-next status?

	Origin net.Addr
	ws *websocket.Conn
}

// get session integer from something that can hand them out

func (session *Session) Send (response Message) {
	if err := websocket.JSON.Send(session.ws, response); err != nil {
		log.Printf("websocket from %s err in send: %q\n", session.Origin, err)
		session.Close()
	}
}

func (session *Session) Close () {

	// so, what do we need to clean up?     Are there
	// some goroutines waiting on things...?

	session.closed = true
}

var cluster *inmem.Cluster

func webHandler(ws *websocket.Conn) {

	session := Session{}
	session.Origin = ws.LocalAddr()
	session.ws = ws
	defer session.Close()

	nextSeq := 0

    in := Message{}
	for {
		in.Seq = nextSeq
		in.Data = nil
		if err := websocket.JSON.Receive(ws, &in); err != nil {
			if err == io.EOF {
				// do something close stuff, handle go routines?
				return; }
			log.Printf("websocket from %s err in receive: %q\n", session.Origin, err)
			return
		}
		nextSeq = in.Seq + 1
		fmt.Printf("Received: %q\n", in)



		var url string
		switch in.Op {
		case "read", "overlay", "delete", "stopQuery":
			url = in.Data["_id"].(string)

			if (!session.inMySpace(url)) {
				session.Send(Message{in.Seq, "fail", 
					JSON{"err":"requested URL not on this pod"}})
				return
			}
		}
	

		switch in.Op {
		case "login":
			// For now we don't do authentication
			// Later on, we'll require a token obtained via a direct channel
			session.userId = in.Data["userId"].(string)
			session.pod,_ = cluster.NewPod(session.userId)
			session.Send(Message{in.Seq, "ok", nil});
		case "create":
			create(session, in)
		case "read":
			read(session, in)
		case "overlay":
			overlay(session, in)
		case "delete":
			pdelete(session, in)
		case "startQuery":
			startQuery(session, in)
		case "stopQuery":
			stopQuery(session, in)
		}
	}
}

func create(session Session, in Message) {
	log.Printf("in.Data[_id]", in.Data["_id"])
	urlintf := in.Data["_id"]
	var page *inmem.Page
	if urlintf == nil {
		page = session.pod.NewPage()
	} else {
		url := urlintf.(string)
		podurl := url [:len(session.pod.URL())]
		log.Printf("podurl %q, url %q , path %q", session.pod.URL(),
			url, url[len(session.pod.URL()):])
		if session.pod.URL() == podurl {
			page,_ = session.pod.PageByURL(url, true)
		} else {
			session.Send(Message{in.Seq, "fail", JSON{"err":"requested prefix is in the wrong web space: "+url+" doesnt start with "+podurl}});
			return
		}
	}
	session.Send(Message{in.Seq, "ok", JSON{"_id":page.URL()}});
}

func (session *Session) inMySpace(url string) bool {
	space := session.pod.URL()
	return url[:len(space)] == space
}

func read(session Session, in Message) {
	url := in.Data["_id"].(string)
	page,_ := session.pod.PageByURL(url, false)
	session.Send(Message{in.Seq, "ok", page.AsJSON()})
}


func overlay(session Session, in Message) {
	url := in.Data["_id"].(string)
	page,_ := session.pod.PageByURL(url, false)
	onlyIfMatch := ""
	if x := in.Data["_etag"]; x != nil {
		onlyIfMatch = x.(string)
	} 
	etag, notMatched := page.SetProperties(in.Data, onlyIfMatch)
	if notMatched {
		session.Send(Message{in.Seq, "fail", JSON{"err":"etag not matched"}})
		return
	}
	session.Send(Message{in.Seq, "ok", JSON{"_etag":etag}})
}

func pdelete(session Session, in Message) {
	url := in.Data["_id"].(string)
	page,_ := session.pod.PageByURL(url, false)
	page.Delete()
	session.Send(Message{in.Seq, "ok", nil})
}

func startQuery(session Session, in Message) {	
	q := NewQuery(session, in)
	go q.loop()
	session.Send(Message{in.Seq, "ok", q.page.AsJSON()})
}

func stopQuery(session Session, in Message) {
	url := in.Data["_id"].(string)
	page,_ := session.pod.PageByURL(url, false)
	page.Set("stop", true)    // vs DELETE?   might want to keep stats, etc?
	session.Send(Message{in.Seq, "ok", nil})
}

func main() {
	cluster = inmem.NewInMemoryCluster("http://example.com")
	fmt.Println("Answering on port :8087/_ws")
	http.Handle("/_ws", websocket.Handler(webHandler))
	err := http.ListenAndServe(":8087", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
