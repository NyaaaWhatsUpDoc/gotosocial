package cache

import (
	"time"

	"codeberg.org/gruf/go-cache"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
)

// AccountCache is a wrapper around cache.LookupCache to provide URL and URI lookups for gtsmodel.Account
type AccountCache struct {
	cache cache.LookupCache
}

// NewAccountCache returns a new instantiated AccountCache object
func NewAccountCache() *AccountCache {
	c := cache.NewLookup(cache.LookupCfg{
		RegisterLookups: func(lm *cache.LookupMap) {
			lm.RegisterLookup("uri")
			lm.RegisterLookup("url")
		},
		AddLookups: func(lm *cache.LookupMap, i interface{}) {
			account := i.(*gtsmodel.Account)
			if account.URI != "" {
				lm.Set("uri", account.URI, account.ID)
			}
			if account.URL != "" {
				lm.Set("url", account.URL, account.ID)
			}
		},
		DeleteLookups: func(lm *cache.LookupMap, i interface{}) {
			account := i.(*gtsmodel.Account)
			if account.URI != "" {
				lm.Delete("uri", account.URI)
			}
			if account.URL != "" {
				lm.Delete("url", account.URL)
			}
		},
	})
	if !c.Stop() || !c.Start(time.Second*30) {
		panic("failed starting cache")
	}
	return &AccountCache{cache: c}
}

// GetByID attempts to fetch a account from the cache by its ID, you will receive a copy for thread-safety
func (c *AccountCache) GetByID(id string) (*gtsmodel.Account, bool) {
	v, ok := c.cache.Get(id)
	if !ok {
		return nil, false
	}
	return copyAccount(v.(*gtsmodel.Account)), true
}

// GetByURL attempts to fetch a account from the cache by its URL, you will receive a copy for thread-safety
func (c *AccountCache) GetByURL(url string) (*gtsmodel.Account, bool) {
	v, ok := c.cache.GetBy("url", url)
	if !ok {
		return nil, false
	}
	return copyAccount(v.(*gtsmodel.Account)), true
}

// GetByURI attempts to fetch a account from the cache by its URI, you will receive a copy for thread-safety
func (c *AccountCache) GetByURI(uri string) (*gtsmodel.Account, bool) {
	v, ok := c.cache.GetBy("uri", uri)
	if !ok {
		return nil, false
	}
	return copyAccount(v.(*gtsmodel.Account)), true
}

// Put places a account in the cache, ensuring that the object place is a copy for thread-safety
func (c *AccountCache) Put(account *gtsmodel.Account) {
	if account == nil || account.ID == "" {
		panic("invalid account")
	}
	c.cache.Set(account.ID, copyAccount(account))
}

// copyAccount performs a surface-level copy of account, only keeping attached IDs intact, not the objects.
// due to all the data being copied being 99% primitive types or strings (which are immutable and passed by ptr)
// this should be a relatively cheap process
func copyAccount(account *gtsmodel.Account) *gtsmodel.Account {
	new := *account
	new.AvatarMediaAttachment = nil
	new.HeaderMediaAttachment = nil
	return &new
}
