package cache

import (
	"sync"
	"time"
)

var (
	// once protects from multiple reapers
	once sync.Once

	// caches is the global slice of initialized caches
	caches []*Cache

	// mutex protects the caches slice
	mutex sync.Mutex
)

// reaper is the global cache eviction routine
func reaper() {
	// Reaper ticker, tick-tick!
	tick := time.Tick(
		clockPrecision * 100,
	)

	for {
		// Rest now little CPU,
		// save your cycles...
		<-tick

		mutex.Lock()
		for _, cache := range caches {
			// sweep-out dust items
			cache.sweep()
		}
		mutex.Unlock()
	}
}
