package main

import (
	"sync"
)

type Cache struct {
	sync.Mutex
	codes map[string][]byte
}

func NewCache() *Cache {
	return &Cache{codes: make(map[string][]byte)}
}

func (c *Cache) Get(url string) ([]byte, bool) {
	c.Lock()
	defer c.Unlock()
	code, ok := c.codes[url]
	return code, ok
}

func (c *Cache) Add(url string, code []byte) {
	c.Lock()
	defer c.Unlock()
	c.codes[url] = code
}
