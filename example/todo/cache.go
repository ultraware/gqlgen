package todo

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

type cache struct {
	requested_ids map[interface{}]struct{}
	storage       map[interface{}]interface{}
	loader        loadFunc
}

type loadFunc func(keys []interface{}) []interface{}

func (c *cache) loadAllRequests() {
	if c.requested_ids != nil && len(c.requested_ids) > 0 {
		//collect requested (and missing) id's
		var ids []interface{}
		for id := range c.requested_ids {
			if c.storage[id] == nil {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			return
		}

		//load missing id's
		values := c.loader(ids)
		//store result
		for i, id := range ids {
			c.storage[id] = values[i]
		}
		//clear
		c.requested_ids = make(map[interface{}]struct{})
	}
}

func (c *cache) getItem(ctx context.Context, id interface{}, loader loadFunc) interface{} {
	c.loader = loader
	if c.storage == nil {
		c.storage = make(map[interface{}]interface{})
	}

	//preparing?
	fctx := graphql.GetFieldContext(ctx)
	if fctx.IsPreparing || (fctx.Parent != nil && fctx.Parent.IsPreparing) {
		if c.requested_ids == nil {
			c.requested_ids = make(map[interface{}]struct{})
		}
		if _, ok := c.requested_ids[id]; !ok {
			var empty struct{}
			c.requested_ids[id] = empty
		}

		var dummy struct{}
		return dummy
	}

	//loading
	result := c.storage[id]
	if result == nil {
		c.loadAllRequests()
		//load again (from cache now)
		result = c.storage[id]
	}
	return result
}
