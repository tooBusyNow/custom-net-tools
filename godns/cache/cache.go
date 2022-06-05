package cache

import (
	"sync"
	"time"

	"github.com/google/gopacket/layers"
)

type Cache struct {
	mu            sync.RWMutex
	cacheLivetime time.Duration
	cleanup       time.Duration
	items         map[string]CacheItem
}

type CacheItem struct {
	RR         layers.DNSResourceRecord
	Created    time.Time
	Expiration int64
}

func (ch *Cache) Add(key string, rr layers.DNSResourceRecord, expTime time.Duration) {
	var expiration int64

	if expTime == 0 {
		expTime = ch.cacheLivetime
	}
	if expTime > 0 {
		expiration = time.Now().Add(expTime).UnixNano()
	}
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.items[key] = CacheItem{
		RR:         rr,
		Expiration: expiration,
		Created:    time.Now(),
	}
}

func NewCache(defaultExpiration, cleanupInterval time.Duration) *Cache {
	items := make(map[string]CacheItem)
	cache := Cache{
		items:         items,
		cacheLivetime: defaultExpiration,
		cleanup:       cleanupInterval,
	}

	if cleanupInterval > 0 {
		cache.StartGC()
	}
	return &cache
}

func (ch *Cache) GetItem(key string) (layers.DNSResourceRecord, bool) {
	ch.mu.RLock()
	defer ch.mu.RLock()

	item, found := ch.items[key]
	if !found {
		return layers.DNSResourceRecord{}, false
	}
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			return layers.DNSResourceRecord{}, false
		}
	}
	return item.RR, true
}

func (ch *Cache) DeleteItem(key string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	_, found := ch.items[key]
	if found {
		delete(ch.items, key)
	}
}

func (ch *Cache) StartGC() {
	go ch.GC()
}

func (ch *Cache) GC() {
	for {
		<-time.After(ch.cleanup)
		if ch.items == nil {
			return
		}
		if expKeys := ch.getExpiredKeys(); len(expKeys) != 0 {
			ch.removeExpiredKeys(expKeys)
		}
	}
}

func (ch *Cache) getExpiredKeys() (expKeys []string) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	for k, item := range ch.items {
		if time.Now().UnixNano() > item.Expiration && item.Expiration > 0 {
			expKeys = append(expKeys, k)
		}
	}
	return expKeys
}

func (ch *Cache) removeExpiredKeys(expKeys []string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	for _, k := range expKeys {
		delete(ch.items, k)
	}
}
