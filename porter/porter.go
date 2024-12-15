package porter

// Porter
// API
// Copyright © 2018-2024 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"hash"
	"hash/fnv"
	"runtime"
	"sort"
	"sync/atomic"
	"time"
)

const timePause time.Duration = 10 * time.Millisecond

const (
	stateUnlocked int32 = iota
	stateLocked
)

/*
Porter - regulates access to resources by keys.
*/
type Porter struct {
	data   [(1 << 32) - 1]int32
	hash32 hash.Hash32
}

func New() *Porter {
	return &Porter{
		hash32: fnv.New32a(),
	}
}

/*
Catch - block certain resources. This function will infinitely try to block the necessary resources,
so if the logic of the application using this library contains errors, deadlocks, etc., this can lead to FATAL errors.
*/
func (p *Porter) Catch(keys []string) {
	hashes := p.stringsToHashes(keys)
	for i, hash := range hashes {
		if !atomic.CompareAndSwapInt32(&p.data[hash], stateUnlocked, stateUnlocked) {
			p.throw(hashes[0:i])
			runtime.Gosched()
			time.Sleep(timePause)
		}
	}
}

/*
Throw - frees access to resources. Resources MUST be blocked before this, otherwise using this library will lead to errors.
*/
func (p *Porter) Throw(keys []string) {
	p.throw(p.stringsToHashes(keys))
}

func (p *Porter) throw(hashes []int) {
	for _, hash := range hashes {
		atomic.StoreInt32(&p.data[hash], stateUnlocked)
	}
}

func (p *Porter) stringsToHashes(keys []string) []int {
	out := make([]int, 0, len(keys))

	for _, key := range keys {
		out = append(out, p.stringToHash(key))
	}

	sort.Ints(out)

	return out
}

func (p *Porter) stringToHash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32())
}
