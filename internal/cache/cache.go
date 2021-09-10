package cache

import (
	"context"
	"sync"
	"time"

	"github.com/kpango/fastime"
)

const clockPrecision = time.Millisecond * 100

// cacheClock is the cache-entry clock, used for TTL checking
var cacheClock = fastime.New().StartTimerD(
	context.Background(),
	clockPrecision,
)

type Cache struct {
	cache  map[string]*entry
	evict  EvictHook  // the evict hook is called when an item is evicted from the cache, includes manual delete
	update UpdateHook // the update hook is called when an existing item is updated in the cache
	ttl    time.Duration
	mutex  sync.Mutex
}

// init performs Cache initialization
func (c *Cache) init() {
	c.cache = make(map[string]*entry, 100)
	c.evict = emptyEvict
	c.update = emptyUpdate
	c.ttl = time.Minute * 5
	go c.cleanup()
}

// New returns a new basic cache
func New() *Cache {
	c := Cache{}
	c.init()
	return &c
}

// cleanup performs regular cache sweeps
func (c *Cache) cleanup() {
	for {
		// Rest now little CPU, save your cycles...
		time.Sleep(clockPrecision * 100)

		// Sweep-out dusty items
		c.sweep()
	}
}

// sweep evicts expired items (with callback!) from cache
func (c *Cache) sweep() {
	// Defer in case hook panics
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Fetch current time for TTL check
	now := cacheClock.Now()

	// Sweep the cache for old items!
	for key, item := range c.cache {
		if item.expiry.After(now) {
			c.evict(key, item.value)
			delete(c.cache, key)
		}
	}
}

// SetEvictionCallback sets the eviction callback to the provided hook
func (c *Cache) SetEvictionCallback(hook EvictHook) {
	// Ensure non-nil hook
	if hook == nil {
		hook = emptyEvict
	}

	// Safely set evict hook
	c.mutex.Lock()
	c.evict = hook
	c.mutex.Unlock()
}

// SetUpdatecallback sets the update callback to the provided hook
func (c *Cache) SetUpdateCallback(hook UpdateHook) {
	// Ensure non-nil hook
	if hook == nil {
		hook = emptyUpdate
	}

	// Safely set update hook
	c.mutex.Lock()
	c.update = hook
	c.mutex.Unlock()
}

// SetTTL sets the cache item TTL. Update can be specified to force updates of existing items in
// the cache, this will simply add the change in TTL to their current expiry time
func (c *Cache) SetTTL(ttl time.Duration, update bool) {
	if ttl < clockPrecision*10 && ttl > 0 {
		// A zero TTL means nothing expires,
		// but other small values we check to
		// ensure they won't be lost by our
		// unprecise cache clock
		panic("ttl too close to cache clock precision")
	}

	// Safely update TTL
	c.mutex.Lock()
	diff := ttl - c.ttl
	c.ttl = ttl

	if update {
		// Update existing cache entries
		for _, entry := range c.cache {
			entry.expiry.Add(diff)
		}
	}

	// We're done
	c.mutex.Unlock()
}

// Get fetches the value with key from the cache, extending its TTL
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.Lock()
	value, ok := c.get(key)
	c.mutex.Unlock()
	return value, ok
}

// get is the mutex-unprotected logic for Cache.Get()
func (c *Cache) get(key string) (interface{}, bool) {
	item, ok := c.cache[key]
	if !ok {
		return nil, false
	}
	item.expiry = cacheClock.Now().Add(c.ttl)
	return item.value, true
}

// Put attempts to place the value at key in the cache, doing nothing if
// a value with this key already exists. Returned bool is success state
func (c *Cache) Put(key string, value interface{}) bool {
	c.mutex.Lock()
	success := c.put(key, value)
	c.mutex.Unlock()
	return success
}

// put is the mutex-unprotected logic for Cache.Put()
func (c *Cache) put(key string, value interface{}) bool {
	// If already cached, return
	if _, ok := c.cache[key]; ok {
		return false
	}

	// Create new cached item
	c.cache[key] = &entry{
		value:  value,
		expiry: cacheClock.Now().Add(c.ttl),
	}

	return true
}

// Set places the value at key in the cache. This will overwrite any
// existing value, and call the update callback so. Existing values
// will have their TTL extended upon update
func (c *Cache) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock() // defer in case of hook panic
	c.set(key, value)
}

// set is the mutex-unprotected logic for Cache.Set(), it calls externally-set functions
func (c *Cache) set(key string, value interface{}) {
	item, ok := c.cache[key]
	if ok {
		// call update hook with old+new
		c.update(key, item.value, value)
	} else {
		// alloc new item
		item = &entry{}
		c.cache[key] = item
	}

	// Update the item + expiry
	item.value = value
	item.expiry = cacheClock.Now().Add(c.ttl)
}

// Has checks the cache for a value with key, this will not update TTL
func (c *Cache) Has(key string) bool {
	c.mutex.Lock()
	_, ok := c.cache[key]
	c.mutex.Unlock()
	return ok
}

// Evict deletes a value from the cache, calling the eviction callback
func (c *Cache) Evict(key string) bool {
	// Defer in case hook panics
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if we have item with key
	item, ok := c.cache[key]
	if !ok {
		return false
	}

	// Forcefully evict item
	c.evict(key, item.value)
	delete(c.cache, key)
	return true
}

// entry represents an item in the cache, with
// it's currently calculated expiry time
type entry struct {
	value  interface{}
	expiry time.Time
}
