/*

Provide access via HTTP, along the lines of LDP 1.0

Mirrors ./websocket.go in functionality.

Constucts an appropriate Session and calls the appropite op functions
based on client requests.

(in theory)

*/

package main

import ( 
	"log"
	"fmt"
	"strings"
	"net/http"
	"encoding/json"
	db "github.com/sandhawke/pagestore/inmem"
)

type HAct struct {
	w http.ResponseWriter
	pod *db.Pod
	userId string
	closed bool
}

func (act *HAct) Event(op string, data JSON) {
	// ignored?
	// 
	// at the moment, we have no idea how to send events over
	// this transport.
	//
}

func (act *HAct) Result(data JSON) {
	act.w.Header().Set("Content-Type", "application/json")
	bytes, _ := json.MarshalIndent(data, "", "    ")
	act.w.Write(bytes)
	fmt.Fprintf(act.w, "\n")
	// anything else...?   close?   push?
	act.closed = true
}

// we need to formalize this more at some point.   Maybe
// flags for types of errors?
func (act *HAct) Error(code int16, message string, details JSON) {
	act.w.WriteHeader(int(code))
	fmt.Fprintf(act.w, "error: %s\n\ndetails: %q\n", message, details)
	act.closed = true
}

func (act *HAct) Closed() bool {
	return act.closed
}

func (act *HAct) Pod() *db.Pod {
	return act.pod
}

func (act *HAct) UserId() string {
	return act.userId
}

func httpHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\n");
	log.Printf("Request %q\n", r)

	// pathparts:=strings.Split(r.URL.Path, "/")
	path := r.URL.Path
	url := "http://"+r.Host+path
	pod := cluster.PodByURL(url)
	var userId string
	if pod != nil {
		// very short term, awaking login mechanism
		userId = pod.URL()   
	}

	log.Printf("URL %q\n", url)

	if r.Method == "GET" { 
		r.ParseForm() 
		log.Printf("Args    %q\n", r.Form)
	}

	log.Printf("Checking origin\n");
	if origin := r.Header.Get("Origin"); origin != "" {
		log.Printf("Allowing access from origin: %q\n", origin)
        w.Header().Set("Access-Control-Allow-Origin", origin)
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE, PATCH")
        w.Header().Set("Access-Control-Expose-Headers",
			"Location, ETag")
        w.Header().Set("Access-Control-Allow-Headers",
            "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Wait-For-None-Match")
    }

	act := &HAct{w, pod, userId, false}

	log.Printf("Method  %q\n", r.Method)
	switch r.Method {
	case "DELETE":
		pageDelete(act, url)

	case "GET":
		switch path {

		// case "_login/0.1.2-alpha-sandro/network.html":
		// 	http.ServeFile(w,r,"/sites/fakepods.com/_login/0.1.2-alpha-sandro/network.html");

		case "_trace": 
			// launch a goroutine that copies the recent and ongoing
			// log entries?
		case "_active":
			/*
			if pod == nil { http.NotFound(w,r); return }
			items := make([]interface{},0)
			for path, res := range pod.Resources {
				if res.Data != nil {
					res.Data["_owner"] = podURL
					res.Data["_id"] = podURL+"/"+path
					res.Data["_etag"] = res.LastMod
					items = append(items, res.Data)
				}
			}
			offerJSON(w,r,jsonobj{"_etag":version,"_members":items})
            */
		case "_nearby":
			/*
			items := make([]jsonobj,0)
			for podURL, pod := range pods {
				for path, res := range pod.Resources {
					if res.Data != nil {
						res.Data["_owner"] = podURL
						res.Data["_id"] = podURL+"/"+path
						res.Data["_etag"] = res.LastMod
						if itemPassesFilter(res.Data, filter) {
							items = append(items, res.Data)
						}
					} // else it's non JSON...
				}
			}
			sort.Sort(ById(items))
			offerJSON(w,r,jsonobj{"_etag":version,"_members":items})
            */
		default:
			log.Printf("default");
			
			// hack for now, until we have proper content-type handling
			// and a real Accept parser
			accept := r.Header.Get("Accept");
			log.Printf("Accept: %q", accept)
			if strings.Contains(accept, "text/html") && 
				!strings.Contains(accept, "application/json") {
				focusPage(act, url)
			} else {
				read(act, url)
			}

			/*
			// if accept html and I'm app/j, then serve a skin @@@
			accept := r.Header.Get("Accept");
			log.Printf("Accept: %q", accept)
			if ( strings.Contains(accept, "text/html") && 
				!strings.Contains(accept, "application/json") &&
				res.ContentType == "application/json" &&
				res.Data != nil ) {
				// poorman's con-neg implementation :)
				res.UpdateData()
				w.Header().Set("Content-Type", "text/html")
				bytes, _ := json.MarshalIndent(res.Data, "", "    ")
				fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<title>@data</title>
<script src="http://crosscloud.org/latest/crosscloud.js"></script>
<script>
var appData = 
`)
				w.Write(bytes)
				fmt.Fprintf(w, `;
crosscloud.micropodProvider("%s");
</script>
</head>
<body onload="crosscloud.embed(null, document.body, appData)">
</body>
</html>
`, "http://fakepods.com/"  /* where SHOULD we get this?  )
				return
			}


			w.Header().Set("Content-Type", res.ContentType)

			if res.Data != nil {
				res.UpdateData()
				bytes, _ := json.MarshalIndent(res.Data, "", "    ")
				w.Write(bytes)
				fmt.Fprintf(w, "\n")
			} else {
				_,_ = res.Body.WriteTo(w)
			} 
		} */
	case "HEAD": 
		// this is oddly handled by go.   hrm.
	case "CRASH":
		panic("just testing")
	case "OPTIONS": 
		// needed for CORS pre-flight
		return
	case "PATCH":
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Sorry, not implemented yet\n")
	case "POST":
		if path != "" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "You can only post at the root of the pod\n")
			return
		}
			/*
		if pod == nil { pod = NewPod(podURL) }
		res = new(Resource)
		res.ContentType = r.Header["Content-Type"][0]
		log.Printf("Content type was %q", res.ContentType)
		if semi := strings.Index(res.ContentType, ";"); semi>0 {
			res.ContentType = res.ContentType[0:semi]
		}
		log.Printf("Content type was %q", res.ContentType)
		res.Body.ReadFrom(r.Body)
		log.Printf("Body was %q", res.Body)
		res.LastMod = pod.NextVersion
		pod.NextVersion++
		name := fmt.Sprintf("r%d", pod.ResourceCounter)
		pod.ResourceCounter++
		pod.Resources[name] = res
		changeWasMade()

		location := podURL+"/"+name
		log.Printf("Location assigned: %q", location)
		w.Header().Set("Location", location)
		w.WriteHeader(http.StatusCreated)

		// try parsing?!
		if res.ContentType == "application/json" || res.ContentType == "application/x-www-form-urlencoded" {
			log.Printf("Parsing JSON %q\n", res.Body.String())
			err := json.Unmarshal(res.Body.Bytes(), &res.Data)
			if err != nil {
				log.Println("error:", err)
			}
			log.Printf("%+v", res.Data)
		} */
	case "PUT":
			/*
		if res == nil {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Use POST to the pod URL to create, please\n")
			return
		}
		res.ContentType = r.Header["Content-Type"][0]
		log.Printf("Content type was %q", res.ContentType)
		if semi := strings.Index(res.ContentType, ";"); semi>0 {
			res.ContentType = res.ContentType[0:semi]
		}
		log.Printf("Content type was %q", res.ContentType)
		newBody := bytes.Buffer{}
		newBody.ReadFrom(r.Body)
		res.Body = newBody
		log.Printf("Body was %q", res.Body)
		
		res.LastMod++
		pod.NextVersion++

		if res.ContentType == "application/json" || res.ContentType == "application/x-www-form-urlencoded" {
			log.Printf("Parsing JSON %q\n", res.Body.String())
			err := json.Unmarshal(res.Body.Bytes(), &res.Data)
			if err != nil {
				log.Println("error:", err)
			}
			log.Printf("%+v", res.Data)
		}
		changeWasMade()
*/
	}
	}
}


func offerJSON(w http.ResponseWriter, r *http.Request, frame JSON) {

	// if they'd prefer HTML, maybe format it as HTML or something?

	bytes, _ := json.MarshalIndent(frame, "", "    ")
	w.Write(bytes)
	fmt.Fprintf(w, "\n")
}


func focusPage(act *HAct, url string) {
	
	log.Printf("focusPage() url %q", url)
	page,_ := cluster.PageByURL(url, false)
	if page == nil {
		act.Error(404, "page not found", JSON{})
		return
	}

	title, hasTitle := page.Get("title")
	if !hasTitle {
		title, hasTitle = page.Get("name")
		if !hasTitle {
			title = "data page"
		}
	}
	data := page.AsJSON()
	act.w.Header().Set("Content-Type", "text/html")
	bytes, _ := json.MarshalIndent(data, "", "    ")
	fmt.Fprintf(act.w, `<!DOCTYPE html>
<html>
<head>
<title>%s</title>
<script src="http://crosscloud.org/focus1/crosscloud.js"></script>
<script>
var appData = 
`, title)
	act.w.Write(bytes)
	fmt.Fprintf(act.w, `;
crosscloud.suggestProvider("%s");
</script>
</head>
<body>
<p>JavaScript is required to properly display this datapage.</p>
<script>
crosscloud.displayInApp(appData)
</script>
</body>
</html>
`, thisHubURL)
}
