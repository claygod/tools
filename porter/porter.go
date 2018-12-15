package porter

// Porter
// API
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"sort"
)

type Porter struct {
}

func (p *Porter) Catch(keys []string) {

}

func (p *Porter) Throw(keys []string) {

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
