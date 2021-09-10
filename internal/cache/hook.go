package cache

// EvictHook defines a function hook that can be supplied as an eviction callback
type EvictHook func(key string, value interface{})

func emptyEvict(string, interface{}) {}

// UpdateHook defines a function hook that can be supplied as an update callback
type UpdateHook func(key string, oldValue, newValue interface{})

func emptyUpdate(string, interface{}, interface{}) {}
