package lru

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCacheNewCache(t *testing.T) {
	_, err := NewCache(0, nil)
	if err == nil {
		t.Error("should return error")
	}
}

func TestCache_Add(t *testing.T) {
	cache, _ := NewCache(2, nil)

	cache.Add("1", 1)

	v, ok := cache.Peek("1")
	if v != 1 {
		t.Errorf("unexpected value")
	}
	if !ok {
		t.Error("should be true")
	}

	cache.Add("2", 2)
	v, _ = cache.Peek("2")

	if v != 2 {
		t.Errorf("unexpected value")
	}

	cache.Add("3", 3)
	if ok = cache.Contains("1"); ok {
		t.Error("should be evicted")
	}

	cache.Add("2", 4)
	v, ok = cache.Peek("2")
	if v != 4 || !ok {
		t.Errorf("expexted '2' to have update value 4")
	}

}

func TestCache_Remove(t *testing.T) {
	cache, _ := NewCache(2, nil)

	cache.Remove("1")
	if _, ok := cache.Peek("1"); ok {
		t.Error("expected cache to remain empty")
	}

	cache.Add("1", 1)

	cache.Remove("1")
	if _, ok := cache.Peek("1"); ok {
		t.Error("should be removed")
	}

	cache.Add("1", 10)
	v, ok := cache.Peek("1")
	if !ok || v != 10 {
		t.Errorf("expected '1' to be added with a new value 10")
	}
}

func TestCache_RemoveWhenFull(t *testing.T) {
	cache, _ := NewCache(2, nil)

	cache.Add("1", 1)
	cache.Add("2", 2)
	cache.Remove("1")

	cache.Add("3", 3)

	if _, ok := cache.Peek("1"); ok {
		t.Error("expected '1' to be removed")
	}
	if _, ok := cache.Peek("2"); !ok {
		t.Error("expected '2' to remain in cache")
	}
	if v, ok := cache.Peek("3"); v != 3 || !ok {
		t.Error("expected '3' to be in cache")
	}
}

func TestCache_RemoveWithEmptyKey(t *testing.T) {
	cache, _ := NewCache(2, nil)
	cache.Add("", 1)

	cache.Remove("")
	if _, ok := cache.Peek(""); ok {
		t.Error("should be removed")
	}
}

func TestCache_Clear(t *testing.T) {

	cache, _ := NewCache(2, nil)
	cache.Add("1", 1)
	cache.Add("2", 2)

	cache.Clear()

	if _, ok := cache.Peek("1"); ok {
		t.Error("expected '1' to be removed")
	}
	if _, ok := cache.Peek("2"); ok {
		t.Error("expected '2' to be removed")
	}

	if len(cache.entriesWithTTL) != 0 || cache.Len() != 0 || cache.list.Len() != 0 {
		t.Error("expected cache to have no entries")
	}
}

func TestCache_Get(t *testing.T) {
	cache, _ := NewCache(2, nil)
	cache.Add("1", 1)

	if v, ok := cache.Get("1"); !ok || v != 1 {
		t.Error("expected '1' to be in cache")
	}
	cache.Add("2", 2)

	_, ok := cache.Get("1")
	if !ok {
		t.Error("expected '1' to be in cache")
	}
}

func TestCache_GetFrequentlyItem(t *testing.T) {
	cache, _ := NewCache(3, nil)

	cache.Add("1", 1)
	cache.Add("2", 2)
	cache.Add("3", 3)

	cache.Get("1")
	cache.Add("4", 4)

	if _, ok := cache.Get("1"); !ok {
		t.Error("expected '1' to be in cache")
	}

	if _, ok := cache.Get("2"); ok {
		t.Error("expected '2' to be removed")
	}
}

func TestCache_GetWithEmptyCache(t *testing.T) {
	cache, _ := NewCache(2, nil)

	if _, ok := cache.Get("1"); ok {
		t.Error("expected '1' doesn't to be in cache")
	}
}

func TestCache_GetWithEmptyKey(t *testing.T) {
	cache, _ := NewCache(2, nil)

	cache.Add("", 100)

	if v, ok := cache.Get(""); !ok || v != 100 {
		t.Error("expected empty key to retrieve correct value from cache")
	}
}

func TestCache_GetWithEmptyValue(t *testing.T) {
	cache, _ := NewCache(2, nil)
	cache.Add("1", nil)

	if v, ok := cache.Get("1"); !ok || v != nil {
		t.Error("expected empty value")
	}
}

func TestCache_AddDuplicateKeys(t *testing.T) {
	cache, _ := NewCache(2, nil)

	cache.Add("1", 1)
	cache.Add("1", 2)

	v, ok := cache.Get("1")
	if !ok || v != 2 {
		t.Error("expected '1' to be in cache")
	}
}

func TestCache_AddWithTTL(t *testing.T) {
	cache, _ := NewCache(2, nil)
	cache.AddWithTTL("1", 1, 1*time.Second)
	cache.AddWithTTL("2", 2, 4*time.Second)

	time.Sleep(1 * time.Second)

	if _, ok := cache.Get("1"); ok {
		t.Errorf("expected '1' to be removed from cache by gourotine")
	}
	if _, ok := cache.Get("2"); !ok {
		t.Errorf("expected '2' to be in cache")
	}

}

func TestCase_AddWithShortTTL(t *testing.T) {
	cache, _ := NewCache(2, nil)
	cache.AddWithTTL("1", 1, 5*time.Millisecond)

	if _, ok := cache.Get("1"); !ok {
		t.Errorf("expected '1' to be in cache")
	}
}

func TestCase_AddWithNullTTL(t *testing.T) {
	cache, _ := NewCache(2, nil)
	cache.AddWithTTL("1", 1, 0)
	if _, ok := cache.Get("1"); ok {
		t.Errorf("expected '1' doesn't to be in cache")
	}
}

func TestCase_AddWithTTLTwice(t *testing.T) {
	cache, _ := NewCache(3, nil)

	cache.AddWithTTL("1", 1, 100*time.Millisecond)
	cache.AddWithTTL("2", 2, 2*time.Second)

	time.Sleep(50 * time.Millisecond)

	cache.AddWithTTL("1", 1, 10*time.Second)

	time.Sleep(70 * time.Millisecond)

	if _, ok := cache.Get("1"); !ok {
		t.Errorf("expected '1' to be in cache")
	}

}

func TestCache_SortedEntriesWithTTL(t *testing.T) {
	cache, _ := NewCache(5, nil)

	cache.AddWithTTL("1", 1, 5*time.Second)
	cache.AddWithTTL("2", 2, 2*time.Second)
	cache.AddWithTTL("3", 3, 4*time.Second)

	// Проверяем порядок в массиве entriesWithTTL
	if len(cache.entriesWithTTL) != 3 {
		t.Fatalf("expected 3 entries in entriesWithTTL, got %d", len(cache.entriesWithTTL))
	}

	for i := 1; i < len(cache.entriesWithTTL); i++ {
		if cache.entriesWithTTL[i-1].ExpiresAt.Before(cache.entriesWithTTL[i].ExpiresAt) {
			t.Errorf(
				"entriesWithTTL is not sorted: entry %d (expires at %v) should be after entry %d (expires at %v)",
				i-1, cache.entriesWithTTL[i-1].ExpiresAt, i, cache.entriesWithTTL[i].ExpiresAt,
			)
		}
	}
}

func TestCache_AddWithTTLWhenFull(t *testing.T) {
	cache, _ := NewCache(2, nil)

	cache.AddWithTTL("1", 1, 5*time.Second)
	cache.AddWithTTL("2", 2, 2*time.Second)

	cache.AddWithTTL("3", 3, 4*time.Second)

	_, ok := cache.Get("1")
	if ok || len(cache.entriesWithTTL) == 3 {
		t.Errorf("expected '1' to be removed from cache")
	}

}

func TestCache_AddWithTTLRemove(t *testing.T) {
	cache, _ := NewCache(2, nil)
	cache.AddWithTTL("1", 1, 5*time.Second)

	cache.Remove("1")

	if _, ok := cache.Get("1"); ok {
		t.Errorf("expected '1' to be removed from cache")
	}
}

func TestCache_RemovingByTTL(t *testing.T) {

	cache, _ := NewCache(5, nil)
	cache.AddWithTTL("1", 1, 5*time.Millisecond)
	cache.AddWithTTL("2", 2, 5*time.Millisecond)
	cache.AddWithTTL("3", 3, 5*time.Millisecond)
	cache.AddWithTTL("4", 4, 5*time.Millisecond)
	cache.AddWithTTL("5", 5, 5*time.Millisecond)

	time.Sleep(7 * time.Millisecond)

	if cache.Len() != 0 {
		fmt.Println(cache.Len())
		t.Errorf("expected empty cache")
	}

}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache, _ := NewCache(1000, nil)

	var wg sync.WaitGroup
	numGoroutines := 1000

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			cache.AddWithTTL(fmt.Sprintf("key-%d", id), id, 2*time.Second)
		}(i)
	}

	wg.Wait()

	if len(cache.items) != 1000 {
		t.Errorf("expected cache to contain 1000 entries, got %d", len(cache.items))
	}
}

func TestCache_ConcurrentAddAndRemove(t *testing.T) {
	cache, _ := NewCache(100, nil)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)

		go func(id int) {
			defer wg.Done()
			cache.Add(fmt.Sprintf("key-%d", id), id)
		}(i)

		go func(id int) {
			defer wg.Done()
			cache.Remove(fmt.Sprintf("key-%d", id))
		}(i)
	}

	wg.Wait()

	if cache.Len() >= 50 {
		t.Errorf("expected cache length to be at most 50, got %d", cache.Len())
	}
}

func TestCache_ClearWithTTL(t *testing.T) {
	cache, _ := NewCache(5, nil)
	cache.AddWithTTL("1", 1, 100*time.Second)
	cache.Add("2", 2)

	cache.Clear()
	if _, ok := cache.Get("1"); ok {
		t.Errorf("expected '1' to be removed from cache")
	}
	if _, ok := cache.Get("2"); ok {
		t.Errorf("expected '2' to be removed from cache")
	}
}

func TestCache_removeOldest(t *testing.T) {
	cache, _ := NewCache(2, nil)

	if k, ok := cache.removeOldest(); ok || k != 0 {
		t.Error("expected empty value")
	}

	cache.Add("1", 1)
	cache.Add("2", 2)

	k, ok := cache.removeOldest()
	if ok && k == 1 {
		t.Error("expected entry to remove from cache")
	}

}

func TestCache_Len(t *testing.T) {
	cache, _ := NewCache(5, nil)

	if cache.Len() != 0 {
		t.Errorf("expected cache length to be 0, got %d", cache.Len())
	}

	cache.Add("key1", "value1")
	cache.Add("key2", "value2")

	if cache.Len() != 2 {
		t.Errorf("expected cache length to be 2, got %d", cache.Len())
	}

	cache.Remove("key1")

	if cache.Len() != 1 {
		t.Errorf("expected cache length to be 1, got %d", cache.Len())
	}
}

func TestCache_Cap(t *testing.T) {
	cache, _ := NewCache(5, nil)

	if cache.Cap() != 5 {
		t.Errorf("expected cache capacity to be 5, got %d", cache.Cap())
	}

	cache, _ = NewCache(10, nil)

	if cache.Cap() != 10 {
		t.Errorf("expected cache capacity to be 10, got %d", cache.Cap())
	}
}

func TestCache_Close(t *testing.T) {
	cache, _ := NewCache(5, nil)

	go func() {
		select {
		case <-cache.done:
		case <-time.After(1 * time.Second):
			t.Errorf("goroutine did not exit after cache was closed")
		}
	}()

	cache.Close()

	// Проверяем, что канал закрыт.
	_, ok := <-cache.done
	if ok {
		t.Errorf("expected channel 'done' to be closed")
	}
}

func TestCache_CloseTwice(t *testing.T) {
	cache, _ := NewCache(5, nil)
	cache.Close()
	// Проверяем повторное закрытие (не должно быть паники).
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	cache.Close()
}

func TestCache_WithStructKeysAndValues(t *testing.T) {
	type ComplexKey struct {
		ID   int
		Name string
	}
	type ComplexValue struct {
		Data map[string]int
		List []string
	}

	cache, _ := NewCache(3, nil)

	key := ComplexKey{ID: 1, Name: "Test"}

	value := ComplexValue{
		Data: map[string]int{"a": 1, "b": 2},
		List: []string{"x", "y"},
	}

	cache.Add(key, value)

	if _, ok := cache.Get(key); !ok {
		t.Errorf("expected to retrieve correct struct value for struct key")
	}
}

func TestCache_WithSliceKeys(t *testing.T) {
	cache, _ := NewCache(3, nil)

	key := []int{1, 2, 3}
	value := "slice"

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic when using slices as keys: %v", r)
		}
	}()

	cache.Add(key, value)

	if v, ok := cache.Get(key); !ok || v != value {
		t.Errorf("expected to retrieve correct value for slice key")
	}
}

func TestCache_WithMapValues(t *testing.T) {
	cache, _ := NewCache(3, nil)

	key := "testMap"
	value := map[string]int{"key1": 10, "key2": 20}

	cache.Add(key, value)

	if v, ok := cache.Get(key); !ok {
		t.Errorf("expected to retrieve map value")
	} else if !mapsEqual(v.(map[string]int), value) {
		t.Errorf("retrieved map does not match expected value")
	}
}

// mapsEqual compares two maps for equality.
func mapsEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

func TestCache_WithChannelValues(t *testing.T) {
	cache, _ := NewCache(3, nil)

	key := "channelKey"
	value := make(chan int)

	cache.Add(key, value)

	if v, ok := cache.Get(key); !ok || v != value {
		t.Errorf("expected to retrieve correct channel value")
	}
}

func TestCache_WithNestedStructs(t *testing.T) {
	type InnerStruct struct {
		Details string
	}
	type OuterStruct struct {
		ID   int
		Data InnerStruct
	}

	cache, _ := NewCache(3, nil)

	key := OuterStruct{ID: 42, Data: InnerStruct{Details: "NestedData"}}
	value := "nestedStructValue"

	cache.Add(key, value)

	if v, ok := cache.Get(key); !ok || v != value {
		t.Errorf("expected to retrieve correct value for nested struct")
	}
}
