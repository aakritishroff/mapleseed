package inmem

import (
	"testing"
)

func BenchmarkPodNewPage(b *testing.B) {

	cluster := NewInMemoryCluster()
	s,_ := cluster.NewPod("http://example.com/")

	for i := 0; i < b.N; i++ {
		s.NewPage()
	}
}
