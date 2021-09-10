package cache

import (
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// StatusCache is a wrapper around Cache to provide URL and URI lookups for gtsmodel.Status
type StatusCache struct {
	cache Cache
	uris  map[string]string // map of status URIs -> IDs
	urls  map[string]string // map of status URLs -> IDs
}

// NewStatus returns a new instantiated statusCache object
func NewStatus() *StatusCache {
	c := StatusCache{}
	c.cache.init()
	c.uris = make(map[string]string, len(c.cache.cache))
	c.urls = make(map[string]string, len(c.cache.cache))
	c.SetEvictionCallback(nil)
	c.SetUpdateCallback(nil)
	return &c
}

func (c *StatusCache) SetEvictionCallback(hook EvictHook) {
	// Ensure non-nil hook
	if hook == nil {
		hook = emptyEvict
	}

	// Safely set evict hook
	c.cache.mutex.Lock()
	c.cache.evict = func(key string, value interface{}) {
		// Get account URI + URL
		status := value.(*gtsmodel.Status)
		uri := status.URI
		url := status.URL

		// Call user hook
		hook(key, value)

		// Remove lookups
		delete(c.uris, uri)
		delete(c.urls, url)
	}
	c.cache.mutex.Unlock()
}

func (c *StatusCache) SetUpdateCallback(hook UpdateHook) {
	// Ensure non-nil hook
	if hook == nil {
		hook = emptyUpdate
	}

	// Safely set update hook
	c.cache.mutex.Lock()
	c.cache.update = func(key string, oldValue, newValue interface{}) {
		// If account lookups changed, update
		oldStt := oldValue.(*gtsmodel.Status)
		newStt := newValue.(*gtsmodel.Status)

		if oldStt.URI != newStt.URI {
			// URI is always populated
			delete(c.uris, oldStt.URI)
			c.uris[newStt.URI] = newStt.ID
		}

		if oldStt.URL != newStt.URL {
			// URL is not always populated

			if oldStt.URL != "" {
				// Old account URL not-empty, delete
				delete(c.urls, oldStt.URL)
			}
			if newStt.URL != "" {
				// New account URL not-empty, update
				c.urls[newStt.URL] = newStt.ID
			}
		}

		// Call user hook
		hook(key, oldValue, newValue)
	}
	c.cache.mutex.Unlock()
}

// GetByID attempts to fetch a status from the cache by its ID, you will receive a copy for thread-safety
func (c *StatusCache) GetByID(id string) (*gtsmodel.Status, bool) {
	c.cache.mutex.Lock()
	status, ok := c.getByID(id)
	c.cache.mutex.Unlock()
	return status, ok
}

// GetByURL attempts to fetch a status from the cache by its URL, you will receive a copy for thread-safety
func (c *StatusCache) GetByURL(url string) (*gtsmodel.Status, bool) {
	// Perform safe ID lookup
	c.cache.mutex.Lock()
	id, ok := c.urls[url]

	// Not found, unlock early
	if !ok {
		c.cache.mutex.Unlock()
		return nil, false
	}

	// Attempt status lookup
	status, ok := c.getByID(id)
	c.cache.mutex.Unlock()
	return status, ok
}

// GetByURI attempts to fetch a status from the cache by its URI, you will receive a copy for thread-safety
func (c *StatusCache) GetByURI(uri string) (*gtsmodel.Status, bool) {
	// Perform safe ID lookup
	c.cache.mutex.Lock()
	id, ok := c.uris[uri]

	// Not found, unlock early
	if !ok {
		c.cache.mutex.Unlock()
		return nil, false
	}

	// Attempt status lookup
	status, ok := c.getByID(id)
	c.cache.mutex.Unlock()
	return status, ok
}

// getByID performs an unsafe (no mutex locks) lookup of status by ID, returning a copy of status in cache
func (c *StatusCache) getByID(id string) (*gtsmodel.Status, bool) {
	v, ok := c.cache.get(id)
	if !ok {
		return nil, false
	}
	return copyStatus(v.(*gtsmodel.Status)), true
}

// Set places a status in the cache, ensuring that the object placed is a copy for thread-safety
func (c *StatusCache) Set(status *gtsmodel.Status) {
	// Check for valid input status
	if status == nil || status.ID == "" || status.URI == "" {
		panic("invalid status")
	}

	// Safely set item in cache
	c.cache.mutex.Lock()
	c.cache.set(status.ID, copyStatus(status))
	c.uris[status.URI] = status.ID
	if status.URL != "" {
		// Status URL is not always set
		c.urls[status.URL] = status.ID
	}
	c.cache.mutex.Unlock()
}

// copyStatus performs a surface-level copy of status, only keeping attached IDs intact, not the objects.
// due to all the data being copied being 99% primitive types or strings (which are immutable and passed by ptr)
// this should be a relatively cheap process
func copyStatus(status *gtsmodel.Status) *gtsmodel.Status {
	s := *status
	s.Attachments = nil
	s.Tags = nil
	s.Mentions = nil
	s.Account = nil
	s.InReplyTo = nil
	s.InReplyToAccount = nil
	s.BoostOf = nil
	s.BoostOfAccount = nil
	return &s
}
