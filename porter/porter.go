package porter

// Porter
// API
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
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

type Porter struct {
	data [4294967295]int32
}

func New() *Porter {
	return &Porter{}
}

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
	tempArr := make(map[int]bool)
	for _, key := range keys {
		tempArr[p.stringToHashe(key)] = true
	}
	for key, _ := range tempArr {
		out = append(out, key)
	}
	sort.Ints(out)
	return out
}

func (p *Porter) stringToHashe(key string) int {
	switch len(key) {
	case 0:
		return 0
	case 1:
		return int(uint(key[0]))
	case 2:
		return int(uint(key[1])<<4) + int(uint(key[0]))
	case 3:
		return int(uint(key[2])<<8) + int(uint(key[1])<<4) + int(uint(key[0]))
	default:
		return int(uint(key[3])<<12) + int(uint(key[2])<<8) + int(uint(key[1])<<4) + int(uint(key[0]))
	}
}
