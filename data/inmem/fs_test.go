package inmem

import (
	"testing"
	"os"
	//"log"
)

func TestFS1(t *testing.T) {
	testDir := "./.fsbind-test"
	podurl := "http://foo.example/bar/"
	w := NewInMemoryCluster()
	os.RemoveAll(testDir)
	w.FSBind(testDir)
	p,existed := w.NewPod(podurl)
	if existed {
		t.Error()
	}
	pg,_ := p.NewPage()
	pg.Set("a", "100");
	id := pg.URL()
	w.Flush() // make sure everything's written before we try to re-read it
	w = nil
	p = nil
	pg = nil

	w = NewInMemoryCluster()
	w.FSBind(testDir)
	p,existed = w.NewPod("http://foo.example/bar/")
	if !existed {
		t.Error("expected p to exist, but it didn't")
	}
	pg,created := w.PageByURL(id, false)
	if pg == nil {
		t.Error()
	}
	if created {
		t.Error()
	}
	if pg.GetDefault("a", nil) != "100" {
		t.Error()
	}
	os.RemoveAll(testDir)
}


func BenchmarkSetYesChangeWITHFS(b *testing.B) {
	testDir := "./.fsbind-test"
	podurl := "http://foo.example/bar/"
	w := NewInMemoryCluster()
	os.RemoveAll(testDir)
	w.FSBind(testDir)
	pod,existed := w.NewPod(podurl)
	if existed {
		b.Error()
	}

	p,_ := pod.NewPage()
	v := 1;
	p.Set("a", v)

	b.ResetTimer() // rmdir might have taken a long time

	for i := 0; i < b.N; i++ {
		p.Set("a",i)
	}
	w.Flush()
}
func BenchmarkSetYesChangeWITHFSAndFlush(b *testing.B) {
	testDir := "./.fsbind-test"
	podurl := "http://foo.example/bar/"
	w := NewInMemoryCluster()
	os.RemoveAll(testDir)
	w.FSBind(testDir)
	pod,existed := w.NewPod(podurl)
	if existed {
		b.Error()
	}

	p,_ := pod.NewPage()
	v := 1;
	p.Set("a", v)

	b.ResetTimer() // rmdir might have taken a long time

	for i := 0; i < b.N; i++ {
		p.Set("a",i)
		w.Flush()
	}
}
