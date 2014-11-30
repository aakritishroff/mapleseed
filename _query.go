package main

import (
	"github.com/sandhawke/inmem/db"
	"sort"
	"time"
)

type Query struct {
	act Act
	seq int
	page *db.Page
	filter db.PageFilter
	sortKey func (page *db.Page) string
	limit uint32
}

func StartQuery(act Act, in InMessage) (q Query) {
	q = Query{}
	q.page = session.pod.NewPage()  //  IN WHAT CONTAINER?
	q.page.Set("isQuery", true)
	q.act = act
	q.seq = in.Seq
	q.constructFilter(in.Data["filter"].(map[string]interface{}))

	// we COULD make the limit come from the page, so it can
	// be tweaked during execution...    :-)
	if x := in.Data["limit"]; x != nil {
		q.limit = uint32(x.(float64))
	}

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

func (query *Query) added(page *db.Page) {
	// at some point add a projection filter to limit the properties
	query.act.Send(OutMessage{query.seq,"add",page.AsJSON()})
}
func (query *Query) removed(page *db.Page) {
	query.act.Send(OutMessage{query.seq,"remove",JSON{"_id":page.URL()}})
}
func (query *Query) searchComplete() {
	query.act.Send(OutMessage{query.seq,"searchComplete",nil})
}


func (query *Query) loop() {

	old := make([]Result,0)   // start with "old" being the empty set

	for {

		// get the modcount before we start; this way we'll re-run the 
		// query if any mods have happened since we started.  There might
		// have been mods early in our traversal -- we don't lock the whole
		// cluster while doing our search
		modCount := cluster.ModCount()

		new := query.runOnce()

		// symmetric set difference on sorted lists
		i:=0
		j:=0
	loop:
		for {

			if query.stopped() {
				query.act.Send(Message{query.seq, "stopped",nil})
				return
			}

			switch {
			case i == len(old) && j == len(new):
				break loop
			case i == len(old):
				query.added(new[j].page)
				j++
			case j == len(new):
				query.removed(old[i].page)
				i++
			case old[i].key == new[j].key:
				i++
				j++
			case old[i].key < new[j].key:
				query.removed(old[i].page)
				i++
			default:
				query.added(new[j].page)
				j++
			}
		}
		query.searchComplete()

		// actually value should come from the query.Page
		duration := query.page.GetDefault("sleepSeconds", 0.05)
		time.Sleep(time.Duration(duration.(float64)))

		cluster.WaitForModSince(modCount)
	}
}


func (query *Query) runOnce() ([]Result) {

	// Lame version for now -- linear search of the cluster,
	// no indexing, no link-following

	rawresult := make([]*db.Page,0)
	cluster.CollectMatchingPages(query.filter, rawresult)

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
	if query.limit != 0 {
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
	// log.Printf("\n\n- comparing %q and %q\n", page, filter);
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

				vmap := v.(JSON)
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
	return true;
}



// to a run-once version?

