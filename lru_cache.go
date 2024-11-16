package lru

import (
	"LRUCache/internal"
	"fmt"
	"github.com/mitchellh/hashstructure/v2"
	"sync"
	"time"
)

const noEvictionTTL = time.Hour * 24 * 365 * 10

type EvictCallback func(key, value any)

type Cache struct {
	size    int
	list    *internal.LruList
	items   map[uint64]*internal.Entry
	onEvict EvictCallback

	mu   sync.Mutex
	done chan struct{}

	entriesWithTTL []*internal.Entry
	ticker         *time.Ticker
}

func NewCache(size int, onEvict EvictCallback) (*Cache, error) {
	if size <= 0 {
		return nil, fmt.Errorf("size must be greater than zero")
	}

	ticker := time.NewTicker(5 * time.Millisecond)

	res := &Cache{
		size:    size,
		list:    internal.NewList(),
		items:   make(map[uint64]*internal.Entry),
		onEvict: onEvict,
		done:    make(chan struct{}),
		ticker:  ticker,
	}

	go res.startEviction()

	return res, nil
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, v := range c.items {
		if c.onEvict != nil {
			c.onEvict(k, v)
		}
		delete(c.items, k)
	}

	c.entriesWithTTL = []*internal.Entry{}

	c.list.Init()
}

func (c *Cache) Add(key, value any) {

	hash, err := hasher(key)
	if err != nil {
		c.onEvict(key, err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[hash]; ok {
		c.list.MoveToFront(ent)
		ent.Value = value
		return
	}

	ent := c.list.PushFront(key, value, time.Now().Add(noEvictionTTL))

	c.items[hash] = ent

	evict := c.size < c.list.Len()
	if evict {
		c.removeOldest()
	}
}

func (c *Cache) Get(key any) (val any, ok bool) {
	hash, err := hasher(key)
	if err != nil {
		c.onEvict(key, err)
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[hash]; ok {
		if time.Now().After(ent.ExpiresAt) {
			fmt.Println(ent.Key, ent.ExpiresAt)
			c.list.Remove(ent)
			delete(c.items, hash)
			c.removeFromEntries(ent)
			return nil, false
		}
		c.list.MoveToFront(ent)
		return ent.Value, true
	}

	return nil, false
}

func (c *Cache) AddWithTTL(key, value any, ttl time.Duration) {
	if ttl == time.Duration(0) {
		return
	}

	TTL := time.Now().Add(ttl)

	hash, err := hasher(key)
	if err != nil {
		c.onEvict(key, err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[hash]; ok {
		c.list.MoveToFront(ent)
		ent.Value = value
		ent.ExpiresAt = TTL
		return
	}

	ent := c.list.PushFront(key, value, TTL)
	c.items[hash] = ent

	c.insertSorted(ent)

	evict := c.size < c.list.Len()
	if evict {
		c.removeOldest()
	}

}

func (c *Cache) Remove(key any) {

	hash, err := hasher(key)
	if err != nil {
		c.onEvict(key, err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[hash]; ok {
		c.list.Remove(ent)
		delete(c.items, hash)
		c.removeFromEntries(ent)
	}

}

func (c *Cache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		// Канал закрыт, ничего не делаем.
	default:
		close(c.done)
	}

}

func (c *Cache) removeOldest() (key any, ok bool) {

	lastEntry := c.list.Last()
	if lastEntry != nil {
		c.list.Remove(lastEntry)

		hash, err := hasher(lastEntry.Key)
		if err != nil {
			return 0, false
		}
		delete(c.items, hash)

		for idx, ent := range c.entriesWithTTL {
			if ent == lastEntry {
				c.entriesWithTTL = append(c.entriesWithTTL[:idx], c.entriesWithTTL[idx+1:]...)
			}
		}

		return lastEntry.Key, true
	}
	return 0, false
}

func (c *Cache) Contains(key any) (ok bool) {
	hash, err := hasher(key)

	if err != nil {
		c.onEvict(key, err)
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok = c.items[hash]

	return ok

}

func (c *Cache) Peek(key any) (val any, ok bool) {

	hash, err := hasher(key)
	if err != nil {
		c.onEvict(key, err)
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var ent *internal.Entry
	ent, ok = c.items[hash]
	if ok {
		return ent.Value, true
	}

	return nil, false
}

// startEviction запускает проверку на устаревшие записи
func (c *Cache) startEviction() {
	for {
		select {
		case <-c.ticker.C:
			c.deleteExpired()
		case <-c.done:
			c.ticker.Stop()
			return
		}
	}
}

// deleteExpired удаляет все устаревшие записи, начиная с конца для оптимизации
func (c *Cache) deleteExpired() {
	c.mu.Lock()

	if len(c.entriesWithTTL) > 0 {
		now := time.Now()

		timeToExpire := time.Until(c.entriesWithTTL[0].ExpiresAt)

		if timeToExpire > 0 {
			c.mu.Unlock()
			time.Sleep(timeToExpire)
			c.mu.Lock()
		}

		for len(c.entriesWithTTL) > 0 {
			lastEntry := c.entriesWithTTL[len(c.entriesWithTTL)-1]
			if lastEntry.ExpiresAt.After(now) {
				break
			}

			hash, _ := hasher(lastEntry.Key)

			delete(c.items, hash)
			c.list.Remove(lastEntry)
			c.entriesWithTTL = c.entriesWithTTL[:len(c.entriesWithTTL)-1]
		}
	}
	c.mu.Unlock()
}

// insertSorted вставляет запись в отсортированный слайс по убыванию ExpiresAt
func (c *Cache) insertSorted(newEntry *internal.Entry) {
	low, high := 0, len(c.entriesWithTTL)
	for low < high {
		mid := (low + high) / 2
		if (c.entriesWithTTL)[mid].ExpiresAt.After(newEntry.ExpiresAt) {
			low = mid + 1
		} else {
			high = mid
		}
	}

	// Вставляем новый элемент на нужную позицию
	c.entriesWithTTL = append(c.entriesWithTTL, nil)
	copy(c.entriesWithTTL[low+1:], c.entriesWithTTL[low:])
	c.entriesWithTTL[low] = newEntry
}

func (c *Cache) removeFromEntries(entry *internal.Entry) {
	for idx, ent := range c.entriesWithTTL {
		if ent == entry {
			c.entriesWithTTL = append(c.entriesWithTTL[:idx], c.entriesWithTTL[idx+1:]...)
		}
	}
}

func (c *Cache) Len() int {
	return len(c.items)
}

func (c *Cache) Cap() int {
	return c.size
}

func hasher(key any) (uint64, error) {
	hash, err := hashstructure.Hash(key, hashstructure.FormatV2, nil)
	if err != nil {
		return 0, err
	}

	return hash, nil
}
