//go:generate go run ../../testdata/gqlgen.go

package todo

import (
	"context"
	"errors"
	"fmt"
	"time"

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
			nexts: []*Next2{
				{ID: 101, Text: `Next 1`, More: &More3{ID: 101, Text: `More 1`}},
				{ID: 102, Text: `Next 2`, More: &More3{ID: 102, Text: `More 2`}},
				{ID: 103, Text: `Next 3`, More: &More3{ID: 103, Text: `More 3`}},
				{ID: 104, Text: `Next 4`, More: &More3{ID: 104, Text: `More 4`}},
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

type resolvers struct {
	todos []*Todo
	nexts []*Next2

	lastID int
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

func (r *resolvers) Next2() Next2Resolver {
	return (*QueryResolver)(r)
}

func (r *resolvers) Sub() SubResolver {
	return (*QueryResolver)(r)
}

type QueryResolver resolvers

func (r *QueryResolver) Todo(ids []int) []*Todo {
	fmt.Print(`DB.getTodos: `, ids, ` `)
	time.Sleep(220 * time.Millisecond)

	var result []*Todo
	for _, key := range ids {
		found := false
		for _, todo := range r.todos {
			if todo.ID == key {
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

func (r *QueryResolver) Todos() []*Todo {
	fmt.Print(`DB.getAllTodos `)
	time.Sleep(220 * time.Millisecond)
	return r.todos
}

func (r *QueryResolver) Sub(objs []*Todo) []*Sub {
	fmt.Print(`DB.getSubs: `, objs, ` `)
	time.Sleep(110 * time.Millisecond)

	results := []*Sub{}
	for _, key := range objs {
		found := false
		for _, todo := range r.todos {
			if todo.ID == key.ID {
				results = append(results, todo.Sub)
				found = true
				break
			}
		}
		if !found {
			results = append(results, nil)
		}
	}
	return results
}

func (r *QueryResolver) Next2(objs []*Sub) []*Next2 {
	fmt.Print(`DB.getNext: `, objs, ` `)
	time.Sleep(110 * time.Millisecond)

	result := []*Next2{}
	for _, key := range objs {
		found := false
		for _, next := range r.nexts {
			if next.ID == key.ID {
				result = append(result, next)
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

func (r *QueryResolver) More(objs []*Next2) []*More3 {
	fmt.Print(`DB.getMore: `, objs, ` `)
	time.Sleep(110 * time.Millisecond)

	result := []*More3{}
	for _, key := range objs {
		found := false
		for _, next := range r.nexts {
			if next.ID == key.ID {
				result = append(result, next.More)
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

func (r *QueryResolver) LastTodo(ctx context.Context) (*Todo, error) {
	if len(r.todos) == 0 {
		return nil, errors.New("not found")
	}
	return r.todos[len(r.todos)-1], nil
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
