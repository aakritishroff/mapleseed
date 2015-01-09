package abstract

import (
	"testing"
)

// shared with other benchmark files
const whamNumPlayers = 1

func BenchmarkWhamCB(b *testing.B) {
	WhackAMole(b.N)
}

func TestWhackAMole(t *testing.T) {
	WhackAMole(10000)
	// WhackAMole(3000000)   //  about 100k per sec
}

func maybeReply(pod Pod, page Page, n int) {
	if page.GetDefault("isMole", false).(bool) {
		reply(pod, page, n)
	}
}

func reply(pod Pod, page Page, n int) {
	trace("player %02d saw mole", n)
	pod.NewPage(JSON{"seenBy": n, "moleSeen": page.URL()})
	trace("player %02d replied %q", n, page.URL())
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
	for p := 0; p < whamNumPlayers; p++ {
		runPlayer(stop, pod, p)
	}

	seen := 0

	// only works for one player...   the others will see it as well...

	cb := func(pagei interface{}) {
		page := pagei.(Page)
		trace("Mole notices %q", page.URL())
		if _, ok := page.Get("moleSeen"); ok {
			trace("Some mole was seen")
			seenBy, ok := page.GetDefault("seenBy", -1).(int)
			if !ok {
				panic("seenBy failed")
			}
			trace("SEEN BY %d", seenBy)
			score[seenBy]++
			seen++
		}
	}

	pod.AddCallback(&cb)

	for i := 0; i < N; i++ {
		trace("raising mole %d", i)
		currentMole, _ := pod.NewPage(JSON{"isMole": true, "moleIndex": i})
		if currentMole == nil {
			panic("mole nil")
		}
		trace("page created %s", currentMole.URL())
		trace("i=%d seen=%d", i, seen)
		if seen != i+1 {
			panic("mole should have been seen by now")
		}
		currentMole.Delete()
		if !currentMole.Deleted() {
			panic("mole should have been deleted via callbacks")
		}
	}

	// log.Printf("Score: %s", score)
}
