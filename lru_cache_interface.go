package lru

import "time"

type ICache interface {
	Cap() int
	Len() int
	Clear()
	Add(key, value any)
	AddWithTTL(key, value any, ttl time.Duration)
	Get(key any) (value any, ok bool)
	Remove(key any)
}
