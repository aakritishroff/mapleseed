/*

The various ops (operations) that a client might request of us,
abstracted away from the communication channel used to make that
request.   These ops are called by ./websocket.go, ./http.go, and
maybe other things (like ./op_test.go).

The operations are basically CRUD + Query, but set up for being on the
Web and for real-time data sync.

*/

package main

import (
	"log"
	"regexp"
	"fmt"
	//"github.com/sandhawke/inmem/db"
)

/*

   An "Act" is a single request-response interaction.  Unlike typical
   HTTP, we support multiple responses to one request, with send and
   sendFinal.  So the "act" parameter is a lot like the (req, res)
   parameters one often passes around, but it's more abstract since
   this isn't just for HTTP.

   I was calling it "Interaction" but that was quite long.  Think of
   it like one act in a many-act play, perhaps.  Or just short for
   "Interaction".

*/
type Act interface {
	Send(op string, data JSON) // when streaming results back
	SendFinal(op string, data JSON)
	Error(code uint32, message string)
	// func pod() *db.Pod // wont change but might be null
	// func clientId() string // == pod.URL() if client owns this pod [multi?]
	// func clientIP() ?
	// func origin() string // domain name of source of browser code
}

var validPodname *regexp.Regexp
func init() {
	validPodname = regexp.MustCompile("^[a-z][a-z0-9]*$")
}

func createPod(act Act, name string) {
	
	if validPodname.MatchString(name) {
		podurl := fmt.Sprintf(podURLTemplate, name)
		pod, existed := cluster.NewPod(podurl)
		if existed {
			act.Error(0, "Pod name already taken")
		} else {
			act.SendFinal("ok", JSON{"_id":pod.URL()})
		}
	} else {
		act.Error(0, "Invalid pod name syntax")
	}
}




// if options are not specified, they'll have "zero" values
type CreationOptions struct {
	inContainer string   // for now, this is the pod URL
	suggestedName string // NOT IMPL
	requiredId string   // NOT IMPL
	initialData JSON   // NOT IMPL
	isConstant bool   // NOT IMPL
}

func create(act Act, options CreationOptions) {

	log.Printf("create() options %q", options)

	pod := cluster.PodByURL(options.inContainer)
	if (pod == nil) {
		act.Error(0, "No such container")
		return
	}
	page,etag := pod.NewPage()
	
	// TODO should set the init value WHILE IT'S LOCKED.

	act.SendFinal("ok", JSON{"_id":page.URL(), "_etag":etag})
	

	/*
      ADD:
           act.tmpIdMap()

           and allow ids like  tmp:whatever
           which get replaced (skolemized) during create,
              so you can send graphs, and send related
              resources in a pipeline, without RTT for each
            

	log.Printf("in.Data[_id]", in.Data["_id"])
	urlintf := in.Data["_id"]
	var page *db.Page
	if urlintf == nil {
		page = act.pod.NewPage()
	} else {
		url := urlintf.(string)
		podurl := url [:len(act.pod.URL())]
		log.Printf("podurl %q, url %q , path %q", act.pod.URL(),
			url, url[len(act.pod.URL()):])
		if act.pod.URL() == podurl {
			page,_ = act.pod.PageByURL(url, true)
		} else {
			act.Send(Message{in.Seq, "fail", JSON{"err":"requested prefix is in the wrong web space: "+url+" doesnt start with "+podurl}});
			return
		}
	}
	act.Send(Message{in.Seq, "ok", JSON{"_id":page.URL()}});

   */
}


func read(act Act, url string) {
	log.Printf("read() url %q", url)
	page,_ := cluster.PageByURL(url, false)
	if page == nil {
		act.Error(404, "page not found")
		return
	}
	act.SendFinal("ok", page.AsJSON())
}

func update(act Act, url string, onlyIfMatch string, data JSON) {
	log.Printf("update() url %q, etag %q, data %q", url, onlyIfMatch, data)
	page,_ := cluster.PageByURL(url, false)
	log.Printf("0");
	etag, notMatched := page.SetProperties(data, onlyIfMatch)
	if notMatched {
		act.Error(409, "etag not matched")
		return
	}
	act.SendFinal("ok", JSON{"_etag":etag})
}

// (delete is a golang keyword, so we'll use pageDelete instead)
func pageDelete(act Act, url string) {
	page,_ := cluster.PageByURL(url, false)
	page.Delete()
	act.SendFinal("ok", JSON{})
}

/*
func startQuery(act Act, in Message) {	
	q := NewQuery(act, in)
	go q.loop()
	act.Send(Message{in.Seq, "ok", q.page.AsJSON()})
}

func stopQuery(act Act, in Message) {
	url := in.Data["_id"].(string)
	page,_ := act.pod.PageByURL(url, false)
	page.Set("stop", true)    // vs DELETE?   might want to keep stats, etc?
	act.Send(Message{in.Seq, "ok", nil})
}


func (act *Act) inMySpace(url string) bool {
	space := act.pod.URL()
	return url[:len(space)] == space
}
*/
