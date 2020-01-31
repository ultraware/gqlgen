//go:generate go run ../../testdata/gqlgen.go

package todo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/mitchellh/mapstructure"
)

var you = &User{ID: 1, Name: "You"}
var them = &User{ID: 2, Name: "Them"}

func getUserId(ctx context.Context) int {
	if id, ok := ctx.Value("userId").(int); ok {
		return id
	}
	return you.ID
}

func New() Config {
	c := Config{
		Resolvers: &resolvers{
			todos: []*Todo{
				{ID: 1, Text: "A todo not to forget", Done: false, owner: you, Sub: &Sub{ID: 101, Text: `Sub 1`}},
				{ID: 2, Text: "This is the most important", Done: false, owner: you, Sub: &Sub{ID: 102, Text: `Sub 2`}},
				{ID: 3, Text: "Somebody else's todo", Done: true, owner: them, Sub: &Sub{ID: 103, Text: `Sub 3`}},
				{ID: 4, Text: "Please do this or else", Done: false, owner: you, Sub: &Sub{ID: 104, Text: `Sub 4`}},
			},
			lastID: 4,
		},
	}
	// c.Directives.HasRole = func(ctx context.Context, obj interface{}, next graphql.Resolver, role Role) (interface{}, error) {
	// 	switch role {
	// 	case RoleAdmin:
	// 		// No admin for you!
	// 		return nil, nil
	// 	case RoleOwner:
	// 		ownable, isOwnable := obj.(Ownable)
	// 		if !isOwnable {
	// 			return nil, fmt.Errorf("obj cant be owned")
	// 		}

	// 		if ownable.Owner().ID != getUserId(ctx) {
	// 			return nil, fmt.Errorf("you dont own that")
	// 		}
	// 	}

	// 	return next(ctx)
	// }
	// c.Directives.User = func(ctx context.Context, obj interface{}, next graphql.Resolver, id int) (interface{}, error) {
	// 	return next(context.WithValue(ctx, "userId", id))
	// }
	return c
}

type cache struct {
	requested_ids map[interface{}]struct{}
	storage       map[interface{}]interface{}
}

type loadFunc func(keys []interface{}) []interface{}

func (c *cache) getItem(ctx context.Context, id interface{}, loader loadFunc) interface{} {
	//preparing?
	fctx := graphql.GetFieldContext(ctx)
	if fctx.DoPrepare || (fctx.Parent != nil && fctx.Parent.DoPrepare) {
		if c.requested_ids == nil {
			c.requested_ids = make(map[interface{}]struct{})
		}
		if _, ok := c.requested_ids[id]; !ok {
			var empty struct{}
			c.requested_ids[id] = empty
		}

		fctx.DoPrepare = false
		fctx.Parent.DoPrepare = false

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

type resolvers struct {
	todos  []*Todo
	lastID int

	tokenCache    cache
	allTokenCache cache
	subCache      cache
}

func (r *resolvers) MyQuery() MyQueryResolver {
	return (*QueryResolver)(r)
}

func (r *resolvers) MyMutation() MyMutationResolver {
	return (*MutationResolver)(r)
}

func (r *resolvers) Todo() TodoResolver {
	return (*QueryResolver)(r)
}

type QueryResolver resolvers

func (r *QueryResolver) getTodos(ids []interface{}) []interface{} {
	fmt.Println(`DB.getTodos: `, ids)
	time.Sleep(220 * time.Millisecond)

	var result []interface{}
	for _, key := range ids {
		id, ok := key.(int)
		if !ok {
			panic("wrong id")
		}

		if id == 666 {
			panic("critical failure")
		}

		found := false
		for _, todo := range r.todos {
			if todo.ID == id {
				result = append(result, todo)
				found = true
				break
			}
		}
		if !found {
			result = append(result, nil)
		}
	}

	return result
}

func (r *QueryResolver) getSubs(ids []interface{}) []interface{} {
	fmt.Println(`DB.getSubs: `, ids)
	time.Sleep(110 * time.Millisecond)

	var result []interface{}
	for _, key := range ids {
		id, ok := key.(int)
		if !ok {
			panic("wrong id")
		}

		found := false
		for _, todo := range r.todos {
			if todo.ID == id {
				result = append(result, todo.Sub)
				found = true
				break
			}
		}
		if !found {
			result = append(result, nil)
		}
	}

	return result
}

type todosWrapper struct {
	todos []*Todo
}

func (r *QueryResolver) getAllTodos(ids []interface{}) []interface{} {
	fmt.Println(`DB.getAllTodos`)
	time.Sleep(220 * time.Millisecond)
	return []interface{}{todosWrapper{todos: r.todos}}
}

func (r *QueryResolver) Todo(ctx context.Context, id int) (*Todo, error) {
	fmt.Println(`get Todo: `, id)
	result := r.tokenCache.getItem(ctx, id, r.getTodos)

	if result == nil {
		return nil, errors.New("not found")
	}
	if todo, ok := result.(*Todo); ok {
		return todo, nil
	}
	return nil, nil
}

func (r *QueryResolver) LastTodo(ctx context.Context) (*Todo, error) {
	if len(r.todos) == 0 {
		return nil, errors.New("not found")
	}
	return r.todos[len(r.todos)-1], nil
}

func (r *QueryResolver) Todos(ctx context.Context) ([]*Todo, error) {
	fmt.Println(`get Todos`)
	var dummy struct{}
	result := r.allTokenCache.getItem(ctx, dummy, r.getAllTodos)

	if todo, ok := result.(todosWrapper); ok {
		return todo.todos, nil
	}
	return nil, nil
}

func (r *QueryResolver) Sub(ctx context.Context, obj *Todo) (*Sub, error) {
	fmt.Println(`get Sub of Todo: `, obj.ID)
	result := r.subCache.getItem(ctx, obj.ID, r.getSubs)

	if todo, ok := result.(*Sub); ok {
		return todo, nil
	}
	return nil, nil
}

type MutationResolver resolvers

func (r *MutationResolver) CreateTodo(ctx context.Context, todo TodoInput) (*Todo, error) {
	newID := r.id()

	newTodo := &Todo{
		ID:    newID,
		Text:  todo.Text,
		owner: you,
	}

	if todo.Done != nil {
		newTodo.Done = *todo.Done
	}

	r.todos = append(r.todos, newTodo)

	return newTodo, nil
}

func (r *MutationResolver) UpdateTodo(ctx context.Context, id int, changes map[string]interface{}) (*Todo, error) {
	var affectedTodo *Todo

	for i := 0; i < len(r.todos); i++ {
		if r.todos[i].ID == id {
			affectedTodo = r.todos[i]
			break
		}
	}

	if affectedTodo == nil {
		return nil, nil
	}

	err := mapstructure.Decode(changes, affectedTodo)
	if err != nil {
		panic(err)
	}

	return affectedTodo, nil
}

func (r *MutationResolver) id() int {
	r.lastID++
	return r.lastID
}
