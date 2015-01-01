package abstract

/* 

  This version does it's own queries, instead of using the query library

  FOR NOW.

*/

import (
	"testing"
)

func maybeReply(pod Pod, page Page, n int) {
	if page.GetDefault("isMole", false).(bool) {
		reply(pod, page, n)
	}
}

func reply(pod Pod, page Page, n int) {
	//log.Printf("player %02d got mole", n)
	pg,_ := pod.NewPage()
	pg.Set("moleSeen", page.URL())
	pg.Set("seenBy", n)
	//log.Printf("player %02d replied %q", n, pg.URL())
}

func runPlayer(pod Pod, n int) chan bool {

	stopchan := make(chan bool)
	listener := make(chan interface{},100)
	pod.AddListener(listener)

	go func() {
		for {
			select {
			case _ = <- stopchan:
				//log.Printf("player %02d got stop", n)
				return
			case pageEvent := <- listener:
				// is this page a mole?  if so, post our own
				page := pageEvent.(Page)
				//log.Printf("player %02d heard %q", n, page.URL())
				maybeReply(pod, page, n) 
			}
		}
	}()
	return stopchan
}

// at some point, the players and the moles should be on different
// pods, connected by the social graph!

func BenchmarkWhackAMole(b *testing.B) {
	WhackAMole(b.N)
}

func TestWhackAMole(t *testing.T) {
	WhackAMole(1)   //  about 10k per sec
}

func WhackAMole(N int) {

	numPlayers := 5

	w := NewWebView()

	pod := NewPod("http://example.com/")
	w.AddPod(pod)

	score := make([]int, numPlayers)
	stop := make([]chan bool, numPlayers)
	for p:=0; p<numPlayers; p++ {
		stop[p] = runPlayer(pod, p)
	}

	// we could run moles in parallel by writing "go" to a channel N
	// times, and letting them race to see who gets it, and pausing
	// when too many are outstanding.  When the channel is closed,
	// they exit.

	listener := make(chan interface{},100)
	pod.AddListener(listener)

	for i := 0; i < N; i++ {
		pg,_ := pod.NewPage()
		pg.Set("isMole", true)	
		//log.Printf("MOLE UP %s", pg.URL())
		// by the time we regain control, it's probably
		// been whacked by all the players
		for {
			found := <- listener
			page := found.(Page)
			if url,ok := page.Get("moleSeen"); ok {
				if url == pg.URL() {
					//log.Printf("MOLE SEEN")
					pg.Delete()
					seenBy := page.GetDefault("seenBy", -1).(int)
					//log.Printf("... BY %d", seenBy)
					score[seenBy]++
					break
				}
			}
		}
	}

	for p:=0; p<numPlayers; p++ {
		close(stop[p])
	}

	//log.Printf("score %s", score)
}

	
