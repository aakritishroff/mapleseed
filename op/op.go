/*

The various ops (operations) that a client might request of us,
abstracted away from the communication channel used to make that
request.   These ops are called by ./websocket.go, ./http.go, and
maybe other things (like ./op_test.go).

The operations are basically CRUD + Query, but set up for being on the
Web and for real-time data sync.

Might be called DatapageNetworkAPI or something like that.

*/

package op

import (
	db "../data/inmem"
	"fmt"
	"log"
	"regexp"
)

type JSON map[string]interface{}

/*

   An "Act" is a single request-response interaction.  Unlike typical
   HTTP, we support multiple responses (mid-request events) on a
   single request, with Event/Result/Error.  So the "act" parameter is
   a lot like the (req, res) parameters one often passes around, but
   it's more abstract since this isn't just for HTTP.

   I was calling it "Interaction" but that was quite long.  Think of
   it like one act in a many-act play, perhaps.  Or just short for
   "Interaction".

*/
type Act interface {
	Closed() bool               // mostly for propagating errors; when closed, just return
	Event(op string, data JSON) // when streaming results back
	Result(data JSON)
	Error(code int16, message string, details JSON)
	//Pod() *db.Pod     // wont change but might be null
	Cluster() *db.Cluster // maybe give a more restrictive interface?
	UserId() string       // == pod.URL() if user owns Pod() being accessed
	// func clientIP() ?
	// func origin() string // domain name of source of browser code
}

func Error1(act Act, message string) {
	act.Error(0, message, JSON{})
}

var validPodname *regexp.Regexp

func init() {
	validPodname = regexp.MustCompile("^[a-z][a-z0-9]*$")
}

func CreatePod(act Act, name string) {

	log.Printf("createPod %q", name)
	if validPodname.MatchString(name) {
		podurl := fmt.Sprintf(act.Cluster().PodURLTemplate, name)
		pod, existed := act.Cluster().NewPod(podurl) // HANGS, some lock
		if existed {
			log.Printf("Pod name %q already taken by %q", name, podurl)
			Error1(act, "Pod name already taken")
		} else {
			log.Printf("created pod %s", pod.URL())
			act.Result(JSON{"_id": pod.URL()})
		}
	} else {
		Error1(act, "Invalid pod name syntax")
	}
}

// if options are not specified, they'll have "zero" values
type CreationOptions struct {
	InContainer   string // for now, this is the pod URL
	SuggestedName string // NOT IMPL
	RequiredId    string // NOT IMPL
	InitialData   JSON   //
	IsConstant    bool   // NOT IMPL
}

func Create(act Act, options CreationOptions) {

	log.Printf("create() options %q", options)

	if options.InContainer == "" {
		options.InContainer = act.UserId()
	}

	pod := act.Cluster().PodByURL(options.InContainer)
	if pod == nil {
		Error1(act, "No such container")
		return
	}
	page, etag := pod.NewPage()

	// TODO should set the init value WHILE IT'S LOCKED.
	etag, _ = page.SetProperties(options.InitialData, "")
	log.Printf("InitialData was %q", options.InitialData)
	log.Printf("now  %q", page.Properties())

	act.Result(JSON{"_id": page.URL(), "_etag": etag})
	val, ok := page.Get("isPublic")
	if !ok { //if property not set, add owner to readers.
		createACL(page, act.UserId(), false) //creates private ACL.
	} else {
		if val.(bool) == false {
			createACL(page, act.UserId(), false) //add owner to readers
		}
	}

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

func Read(act Act, url string) {
	log.Printf("read() url %q", url)
	page, _ := act.Cluster().PageByURL(url, false)
	if page == nil {
		act.Error(404, "page not found", JSON{})
		return
	}
	if !isReadable(act.UserId(), page) {
		act.Error(550, "Permission Denied. Not in list of page's readers", JSON{})
		return
	}
	act.Result(page.AsJSON())
}

func Update(act Act, url string, onlyIfMatch string, data JSON) {
	log.Printf("update() url %q, etag %q, data %q", url, onlyIfMatch, data)
	page, _ := act.Cluster().PageByURL(url, false)
	if page == nil {
		act.Error(404, "page not found", JSON{})
		return
	}
	log.Printf("0")
	etag, notMatched := page.SetProperties(data, onlyIfMatch)
	if notMatched {
		act.Error(409, "etag not matched", JSON{})
		return
	}
	act.Result(JSON{"_etag": etag})
}

func Delete(act Act, url string) {
	page, _ := act.Cluster().PageByURL(url, false)
	if page == nil {
		act.Error(404, "page not found", JSON{})
		return
	}
	page.Delete()
	act.Result(JSON{})
}

func StartQuery(act Act, options QueryOptions) {
	if options.InContainer == "" {
		options.InContainer = act.UserId()
	}
	q := NewQuery(act, options)
	if q == nil {
		return
	}
	go q.loop()
	if act.Closed() {
		return
	}
	act.Event("QueryCreated", q.page.AsJSON())
	// result/error will come much later
}

func StopQuery(act Act, url string) {
	page, _ := act.Cluster().PageByURL(url, false)
	if page == nil {
		act.Error(404, "No such query", JSON{})
		return
	}
	page.Set("stop", true) // vs DELETE?   might want to keep stats, etc?
	act.Result(page.AsJSON())
}

/*
func (act *Act) inMySpace(url string) bool {
	space := act.pod.URL()
	return url[:len(space)] == space
}
*/
