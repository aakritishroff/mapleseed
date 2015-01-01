package abstract

/* 

  DEADLOCKs if the channels dont have enough buffering  :-(
  [ or, it did BEFORE .Notify() forked a goroutine ]

*/

import (
	"testing"
)

const chanSize = 0

func BenchmarkWhamChan1P(b *testing.B) {
	chWham(b.N, 1)
}
func BenchmarkWhamChan2P(b *testing.B) {
	chWham(b.N, 2)
}
func BenchmarkWhamChan10P(b *testing.B) {
	chWham(b.N, 10)
}
func BenchmarkWhamChan20P_sc(b *testing.B) {
	chWham(b.N/20, 20)
}

func TestWhamChan1P(t *testing.T) {
	chWham(1000,1)
}
func TestWhamChan2P(t *testing.T) {
	chWham(1000,2)
}
func TestWhamChan10P(t *testing.T) {
	chWham(1000,10)
}






func chmaybeReply(pod Pod, page Page, n int, prevWhack Page) {
	// interestingly we do a double whack.   The first notify
	// will be from before isMole was set, but by the time we
	// get this, isMole will be set, so we'll think both notifies
	// are for the Mole, and they kind of are.  *shrug*
	if page.GetDefault("isMole", false).(bool) {
		trace("player %02d heard MOLE %q", n, page.URL())

		if prevWhack != nil {
			trace("player %02d deleting prevWhack %s", n, prevWhack.URL())
			prevWhack.Delete()
		}

		chreply(pod, page, n, prevWhack)
	}
}

func chreply(pod Pod, page Page, n int, prevWhack Page) {
	trace("player %02d got mole, replying...", n)
	pg,_ := pod.NewPage(JSON{"seenBy":n,"moleSeen":page.URL()})
	trace("player %02d replied %q", n, pg.URL())
	prevWhack = pg
}

func chrunPlayer(stop chan struct{}, pod Pod, n int) {

	listener := make(chan interface{},chanSize)
	pod.AddListener(listener)
	var prev Page

	go func() {
		defer trace("player %02d returning ch", n)
		for {
			select {
			case <- stop:
				trace("player %02d got stop ch", n)
				return
			case pageEvent := <- listener:
				// is this page a mole?  if so, post our own
				page := pageEvent.(Page)
				trace("player %02d heard %q", n, page.URL())
				chmaybeReply(pod, page, n, prev) 
			}
		}
	}()
	return
}

// at some point, the players and the moles should be on different
// pods, connected by the social graph!

func chWham(N int, numPlayers int) {


	w := NewWebView()

	pod := NewPod("http://example.com/")
	w.AddPod(pod)

	// by puting the mole at the head of the queue, we can see the
	// whacks first and remove the mole immediately, so not everyone
	// has to whack it before we even notice.  Heh.  You wish.  These
	// aren't callbacks, they're channels.  It'll be in our channel,
	// first, but we probably wont get scheduled to read it (so we can
	// handle it) until many others have seen the mole and whacked it,
	// filling everyone's queues with their whacks.
	listener := make(chan interface{},chanSize)
	pod.AddListener(listener)


	score := make([]int, numPlayers)
	stop := make(chan struct{})
	for p:=0; p<numPlayers; p++ {
		chrunPlayer(stop, pod, p)
	}

	// we could run moles in parallel by writing "go" to a channel N
	// times, and letting them race to see who gets it, and pausing
	// when too many are outstanding.  When the channel is closed,
	// they exit.

	for i := 0; i < N; i++ {
		pg,_ := pod.NewPage()
		pg.Set("isMole", true)	
		trace("MOLE %d UP %s", i, pg.URL())
		// by the time we regain control, it's probably
		// been whacked by all the players
		for {
			found := <- listener
			page := found.(Page)
			trace("mole heard %q", page.URL())
			if url,ok := page.Get("moleSeen"); ok {
				if url == pg.URL() {
					trace("MOLE SEEN")
					pg.Delete()
					seenBy := page.GetDefault("seenBy", -1).(int)
					trace("... BY %d of %d", seenBy, numPlayers)
					score[seenBy]++
					break
				}
			}
		}
	}
	trace("sending stop")

	close(stop)
	// log.Printf("Score: %s", score)

}

	
