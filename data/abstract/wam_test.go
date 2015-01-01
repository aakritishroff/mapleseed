package abstract

import (
	"testing"
	"log"
)

// shared with other benchmark files
const whamNumPlayers = 1


func trace(template string, args ...interface{}) {
	if false {
		log.Printf(template, args...)
	}
}

func BenchmarkWhackAMole(b *testing.B) {
	WhackAMole(b.N)
}

func TestWhackAMole(t *testing.T) {
	//WhackAMole(300000)   //  about 10k per sec
	WhackAMole(1000)   //  about 10k per sec
}




func maybeReply(pod Pod, page Page, n int) {
	if page.GetDefault("isMole", false).(bool) {
		reply(pod, page, n)
	}
}

func reply(pod Pod, page Page, n int) {
	trace("player %02d saw mole", n)
	pg,_ := pod.NewPage()
	trace("player %02d created reply page %q", n, pg.URL())
	pg.Set("seenBy", n)
	trace("player %02d set seenBy on %q", n, pg.URL())
	pg.Set("moleSeen", page.URL())
	trace("player %02d replied %q", n, pg.URL())
}

func runPlayer(stop chan struct{}, pod Pod, n int) {

	cb := func(pagei interface{}) {
		trace("player %02d saw something", n)
		page := pagei.(Page)
		maybeReply(pod, page, n) 
	}
	pod.AddCallback(&cb)
	trace("player %02d active", n)

}

// at some point, the players and the moles should be on different
// pods, connected by the social graph!

func WhackAMole(N int) {

	w := NewWebView()

	pod := NewPod("http://example.com/")
	w.AddPod(pod)

	score := make([]int, whamNumPlayers)
	stop := make(chan struct{})
	for p:=0; p<whamNumPlayers; p++ {
		runPlayer(stop, pod, p)
	}

	var currentMole Page

	cb := func(pagei interface{}) {
		page := pagei.(Page)
		trace("Mole notices %q", page.URL()) 
		if url,ok := page.Get("moleSeen"); ok {
			trace("Some mole was seen")
			if url == currentMole.URL() {
				trace("THIS MOLE SEEN")
				currentMole.Delete()
				trace("DELETED")
				seenBy,ok := page.GetDefault("seenBy", -1).(int)
				if !ok {
					panic("seenBy failed")
				}
				trace("SEEN BY %d", seenBy)
				score[seenBy]++
			}
		}
	}

	pod.AddCallback(&cb)

	for i := 0; i < N; i++ {
		trace("raising mole %d", i)
		currentMole,_ = pod.NewPage()
		trace("page created %s", currentMole.URL())
		currentMole.Set("isMole", true)	
		trace("mole was raised %s", currentMole.URL())	
		if ! currentMole.Deleted() {
			panic("mole should have been deleted via callbacks")
		}
	}

	
	trace("Score: %s", score)
}

	
