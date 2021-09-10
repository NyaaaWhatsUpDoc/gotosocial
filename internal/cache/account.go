package cache

import (
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// AccountCache is a wrapper around Cache to provide URL and URI lookups for gtsmodel.Account
type AccountCache struct {
	cache Cache             // map of IDs -> cached accounts
	urls  map[string]string // map of account URLs -> IDs
	uris  map[string]string // map of account URIs -> IDs
}

// NewAccount returns a new instantiated AccountCache object
func NewAccount() *AccountCache {
	c := AccountCache{}
	c.cache.init()
	c.uris = make(map[string]string, len(c.cache.cache))
	c.urls = make(map[string]string, len(c.cache.cache))
	c.SetEvictionCallback(nil)
	c.SetUpdateCallback(nil)
	return &c
}

func (c *AccountCache) SetEvictionCallback(hook EvictHook) {
	// Ensure non-nil hook
	if hook == nil {
		hook = emptyEvict
	}

	// Safely set evict hook
	c.cache.mutex.Lock()
	c.cache.evict = func(key string, value interface{}) {
		// Get account URI + URL
		account := value.(*gtsmodel.Account)
		uri := account.URI
		url := account.URL

		// Call user hook
		hook(key, value)

		// Remove lookups
		delete(c.uris, uri)
		delete(c.urls, url)
	}
	c.cache.mutex.Unlock()
}

func (c *AccountCache) SetUpdateCallback(hook UpdateHook) {
	// Ensure non-nil hook
	if hook == nil {
		hook = emptyUpdate
	}

	// Safely set update hook
	c.cache.mutex.Lock()
	c.cache.update = func(key string, oldValue, newValue interface{}) {
		// If account lookups changed, update
		oldAcc := oldValue.(*gtsmodel.Account)
		newAcc := newValue.(*gtsmodel.Account)

		if oldAcc.URI != newAcc.URI {
			// URI is always populated
			delete(c.uris, oldAcc.URI)
			c.uris[newAcc.URI] = newAcc.ID
		}

		if oldAcc.URL != newAcc.URL {
			// URL is not always populated

			if oldAcc.URL != "" {
				// Old account URL not-empty, delete
				delete(c.urls, oldAcc.URL)
			}
			if newAcc.URL != "" {
				// New account URL not-empty, update
				c.urls[newAcc.URL] = newAcc.ID
			}
		}

		// Call user hook
		hook(key, oldValue, newValue)
	}
	c.cache.mutex.Unlock()
}

// GetByID attempts to fetch a account from the cache by its ID, you will receive a copy for thread-safety
func (c *AccountCache) GetByID(id string) (*gtsmodel.Account, bool) {
	c.cache.mutex.Lock()
	account, ok := c.getByID(id)
	c.cache.mutex.Unlock()
	return account, ok
}

// GetByURL attempts to fetch a account from the cache by its URL, you will receive a copy for thread-safety
func (c *AccountCache) GetByURL(url string) (*gtsmodel.Account, bool) {
	// Perform safe ID lookup
	c.cache.mutex.Lock()
	id, ok := c.urls[url]

	// Not found, unlock early
	if !ok {
		c.cache.mutex.Unlock()
		return nil, false
	}

	// Attempt account lookup
	account, ok := c.getByID(id)
	c.cache.mutex.Unlock()
	return account, ok
}

// GetByURI attempts to fetch a account from the cache by its URI, you will receive a copy for thread-safety
func (c *AccountCache) GetByURI(uri string) (*gtsmodel.Account, bool) {
	// Perform safe ID lookup
	c.cache.mutex.Lock()
	id, ok := c.uris[uri]

	// Not found, unlock early
	if !ok {
		c.cache.mutex.Unlock()
		return nil, false
	}

	// Attempt account lookup
	account, ok := c.getByID(id)
	c.cache.mutex.Unlock()
	return account, ok
}

// getByID performs an unsafe (no mutex locks) lookup of account by ID, returning a copy of account in cache
func (c *AccountCache) getByID(id string) (*gtsmodel.Account, bool) {
	v, ok := c.cache.get(id)
	if !ok {
		return nil, false
	}
	return copyAccount(v.(*gtsmodel.Account)), true
}

// Set places a account in the cache, ensuring that the object place is a copy for thread-safety
func (c *AccountCache) Set(account *gtsmodel.Account) {
	// Check for valid input account
	if account == nil || account.ID == "" || account.URI == "" {
		panic("invalid account")
	}

	// Safely set item in cache
	c.cache.mutex.Lock()
	c.cache.set(account.ID, copyAccount(account))
	c.uris[account.URI] = account.ID
	if account.URL != "" {
		// Account URL is not always set
		c.urls[account.URL] = account.ID
	}
	c.cache.mutex.Unlock()
}

// copyAccount performs a surface-level copy of account, only keeping attached IDs intact, not the objects.
// due to all the data being copied being 99% primitive types or strings (which are immutable and passed by ptr)
// this should be a relatively cheap process
func copyAccount(account *gtsmodel.Account) *gtsmodel.Account {
	a := *account
	a.AvatarMediaAttachment = nil
	a.HeaderMediaAttachment = nil
	return &a
}
