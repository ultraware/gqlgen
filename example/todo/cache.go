package todo

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

type cache struct {
	requested_ids map[interface{}]struct{}
	storage       map[interface{}]interface{}
}

type loadFunc func(keys []interface{}) []interface{}

func (c *cache) getItem(ctx context.Context, id interface{}, loader loadFunc) interface{} {
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

	if c.storage == nil {
		c.storage = make(map[interface{}]interface{})
	}

	//loading
	result := c.storage[id]
	if result == nil {
		//collect requested (and missing) id's
		var ids []interface{}
		for id := range c.requested_ids {
			if c.storage[id] == nil {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			ids = append(ids, id)
		}
		//load missing id's
		values := loader(ids)
		//store result
		for i, id := range ids {
			c.storage[id] = values[i]
		}
		//clear
		c.requested_ids = make(map[interface{}]struct{})
		//load again (from cache now)
		result = c.storage[id]
	}
	return result
}
