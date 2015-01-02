package op

import (
    "log"
    "sort"
    "time"
	db "github.com/sandhawke/mapleseed/data/inmem"
)

type QueryOptions struct {
    InContainer string
    Filter JSON
    Watching_AllResults bool
    Watching_Appear     bool
    Watching_Disappear  bool
    Watching_Progress   bool
    Limit uint32
}

type Query struct {
    act Act
    pod *db.Pod
    page *db.Page
    filter db.PageFilter
    sortKey func (page *db.Page) string
    limit uint32
    count uint32
    options QueryOptions
}

func NewQuery(act Act, options QueryOptions) (q *Query) {
    q = &Query{}
    q.options = options // for the ones that don't need processing
    q.pod = act.Cluster().PodByURL(options.InContainer)
    if (q.pod == nil) {
		q = nil;
        act.Error(400, "bad InContainer value", JSON{"valueUsed":options.InContainer});
        return
    }
    q.page,_ = q.pod.NewPage()
    q.page.Set("isQuery", true)
    q.act = act
    q.constructFilter(options.Filter)

    // we COULD make the limit come from the page, so it can
    // be tweaked during execution...    :-)
    q.limit = options.Limit

    // assume no in.Data["sortBy"] for now, so sort by _id
    q.sortKey = func (page *db.Page) string {
        return page.URL()
    }

    return
}

func (query *Query) constructFilter(expr JSON) {

    // should we special case certain exprs to have a simpler
    // filter function?   *shrug*

    query.filter = func (page *db.Page) bool {
        return pagePassesFilter(page, expr)
    }
}

func (query *Query) stopped() bool {
    if query.page.Deleted() { return true}
    return query.page.GetDefault("stop", false).(bool)
}

func (query *Query) differenceFound(different *bool) {
    if (!*different) {
        *different = true
        if query.options.Watching_Progress {
            query.act.Event("Progress",JSON{"percentComplete":float64(0),"results":query.count})
        }
    }
}

func (query *Query) added(page *db.Page) {
    // at some point add a projection filter to limit the properties
	log.Printf("Appear %s", page.URL());
    if query.options.Watching_Appear {
        query.act.Event("Appear",page.AsJSON())
    }
}
func (query *Query) removed(page *db.Page) {
	log.Printf("Disappear %s", page.URL());
    if query.options.Watching_Disappear {
        query.act.Event("Disappear",JSON{"_id":page.URL()})
    }
}
func (query *Query) searchComplete() {
    if query.options.Watching_Progress {
        query.act.Event("Progress",JSON{"percentComplete":float64(100)})
    }
}


func (query *Query) loop() {

    old := make([]Result,0)   // start with "old" being the empty set

	lastActivity := time.Now()
	var lastSentAtModCount uint64

    for {

        // get the modcount before we start; this way we'll re-run the 
        // query if any mods have happened since we started.  There might
        // have been mods early in our traversal -- we don't lock the whole
        // cluster while doing our search
        modCount := query.act.Cluster().ModCount()

        // query.act.Event("progress",JSON{"percentComplete":float64(0)})
        new := query.runOnce()

        log.Printf("Ran query, producing: %q", new)

        // symmetric set difference on sorted lists
        i:=0
        j:=0
        different:=false
		firstTime := true
        log.Printf("%q doing diff, closed? %b", &query, query.act.Closed());
    loop:
        for {

            if query.act.Closed() { return }

            if query.stopped() {
                query.act.Result(query.page.AsJSON())
                return
            }


            // NOTE that this does NOT look for CHANGES in a page,
            // that's supposed to be handled elsewhere (autopull?).
            // This is just about which pages appeared or disappeared
            // from the match (based on the key value)

            switch {
            case i == len(old) && j == len(new):
                break loop
            case i == len(old):
                query.differenceFound(&different)
                query.added(new[j].page)
                j++
            case j == len(new):
                query.differenceFound(&different)
                query.removed(old[i].page)
                i++
            case old[i].key == new[j].key:
                i++
                j++
            case old[i].key < new[j].key:
                query.differenceFound(&different)
                query.removed(old[i].page)
                i++
            default:
                query.differenceFound(&different)
                query.added(new[j].page)
                j++
            }
        }
        if different {
            query.searchComplete()
        }

		// we can't use the 'different' flag, because we care
		// about the values changing as well
        if query.options.Watching_AllResults {
			log.Printf("Computing AllResults")

			propsChanged := false

            pages := make([]JSON,len(new))
            for i, result := range new {
				if result.page.LastModifiedAtClusterModCount() > lastSentAtModCount {
					propsChanged = true
				}
                pages[i] = result.page.AsJSON();
            }
			if propsChanged || different || firstTime {
				firstTime = false
				log.Printf("AllResults count %d", query.count)
				query.act.Event("AllResults", JSON{"results":pages,"fullCount":query.count})
				lastActivity = time.Now()
				lastSentAtModCount = modCount
			}
        } else {
			log.Printf("AllResults not wanted")
		}

        old = new

        // actually value should come from the query.Page
        log.Printf("%q sleeping, closed? %b, different %b", &query, query.act.Closed(), different);
        duration := query.page.GetDefault("sleepSeconds", 0.05)
        time.Sleep(time.Duration(duration.(float64)))

        if query.act.Closed() { return }

        log.Printf("waiting for mod since %d (its %d)", modCount, query.act.Cluster().ModCount());
        query.act.Cluster().WaitForModSince(modCount)
        log.Printf("WAITED, %d != %d", modCount, query.act.Cluster().ModCount());

		// if we're running through this loop, without doing any network
		// activity, we'd never notice if the client went away.  So make
		// sure there's some network activity every few seconds.  (This only
		// happens if we're being triggered to re-evaluate.  If we're not,
		// there's very little cost.)
		now := time.Now()
		if now.Sub(lastActivity).Seconds() > 5 {
			query.act.Event("NetworkCheck", JSON{})
			lastActivity = now
		} 
		
    }
}


func (query *Query) runOnce() ([]Result) {

    // Lame version for now -- linear search of the cluster,
    // no indexing, no link-following

    rawresult := make([]*db.Page,0)
    query.act.Cluster().CollectMatchingPages(query.filter, &rawresult)
    query.count = uint32(len(rawresult))
    //log.Printf("rawresult: %q", rawresult)

    // in theory we could be keeping the collected matching 
    // pages in sorted order, so that we only need to accumate
    // offset+limit of them....      Or maybe there are better
    // ways than that.  Maybe indexing?  :-)

    result := make([]Result,len(rawresult))
    for i,page := range rawresult {
        key := query.sortKey(page)
        result[i] = Result{page,key}
    }
    sort.Sort(ByKey(result))
    if query.limit != 0 && query.count > query.limit {
        //log.Printf("Truncating to %q", query.limit)
        result = result[:query.limit]
    }
    
    return result
}

type Result struct {
    page *db.Page
    key string   //  *shudder* at keeping this, quick-ish hack
    // maybe switch to http://play.golang.org/p/4PmJVi2_7D style?
}

type ByKey []Result
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].key < a[j].key }


func pagePassesFilter(page *db.Page, expr JSON) bool {
    //log.Printf("\n\n- comparing %q and %q\n", page.AsJSON(), expr);
    for k,v := range expr {
        var presentValue, exists  = page.Get(k);
        if exists && presentValue == v {
            // log.Printf("- comparing %q value %q %q: true!", k, v, page[k]);
        } else {
            switch v.(type) {
            case string:
                return false
            case map[string]interface{}:

                // conjoin all the operators, I guess...

                vmap := v.(map[string]interface {})
                if (vmap["$exists"] == true) {
                    //log.Printf("  %q exists?", k)
                    if !exists {
                        return false;
                    } else {
                        //log.Printf("      YES  %q", page[k])
                    }
                } else if (vmap["$exists"] == false) {
                    if exists {
                        return false;
                    } else {
                        // okay
                    }
                }

            default:
                // log.Printf("- comparing %q value %q %q; FALSE (type %q)", k, v, page[k], t);
                return false
            }
        }
    }
    //log.Printf("it's a MATCH");
    return true;
}

