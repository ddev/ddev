// Method outlined on the blog of Karl Seguin
// http://openmymind.net/Do-More-In-Process-Caching/

package cache

import (
	"sync"
)

type Entry interface {
	GetID() string
}

type Cache struct {
	items map[string]Entry
	lock  *sync.RWMutex
}

func New() *Cache {
	return &Cache{
		items: make(map[string]Entry, 1024),
		lock:  new(sync.RWMutex),
	}
}

func (c *Cache) Get(id string) Entry {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.items[id]
}

func (c *Cache) Add(item Entry) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.items[item.GetID()] = item
}

func (c *Cache) Remove(id string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.items, id)
}
