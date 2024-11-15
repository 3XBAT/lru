package internal

import "time"

type Entry struct {
	next, prev *Entry
	List       *LruList
	Key        any
	Value      any
	ExpiresAt  time.Time
}

type LruList struct {
	root Entry // псевдоэлемент
	len  int
}

func (l *LruList) Init() *LruList {
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

func NewList() *LruList { return new(LruList).Init() }

func (l *LruList) Len() int { return l.len }

func (l *LruList) lazyInit() {
	if l.root.next == nil {
		l.Init()
	}
}

func (l *LruList) PushFront(k, v any, expiresAt time.Time) *Entry {
	l.lazyInit()
	return l.insertValue(k, v, expiresAt, &l.root)
}

func (l *LruList) Last() *Entry {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

func (l *LruList) MoveToFront(entry *Entry) {
	if entry.List != l || l.root.next == entry {
		return
	}

	l.move(entry, &l.root)
}

func (l *LruList) insert(entry, at *Entry) *Entry {
	entry.prev = at
	entry.next = at.next
	entry.prev.next = entry
	entry.next.prev = entry
	entry.List = l
	l.len++

	return entry
}

func (l *LruList) insertValue(k, v any, expiresAt time.Time, at *Entry) *Entry {
	return l.insert(&Entry{Value: v, Key: k, ExpiresAt: expiresAt}, at)
}

func (l *LruList) move(entry, at *Entry) {
	if entry == at {
		return
	}

	entry.prev.next = entry.next
	entry.next.prev = entry.prev

	entry.prev = at
	entry.next = at.next
	entry.prev.next = entry
	entry.next.prev = entry

}

func (l *LruList) Remove(entry *Entry) any {
	entry.next.prev = entry.prev
	entry.prev.next = entry.next
	entry.next = nil
	entry.prev = nil
	entry.List = nil
	l.len--

	return entry.Value
}

//func (l *LruList) Top() *Entry {
//	if l.len == 0 {
//		return nil
//	}
//
//	return l.root.next
//}
