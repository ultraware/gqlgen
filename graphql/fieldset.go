package graphql

import (
	"context"
	"fmt"
	"io"
	"sync"
)

type FieldSet struct {
	fields  []CollectedField
	Values  []Marshaler
	delayed []delayedResult
}

type delayedResult struct {
	i int
	f func() Marshaler
}

func NewFieldSet(fields []CollectedField) *FieldSet {
	return &FieldSet{
		fields: fields,
		Values: make([]Marshaler, len(fields)),
	}
}

func (m *FieldSet) Concurrently(i int, f func() Marshaler) {
	m.delayed = append(m.delayed, delayedResult{i: i, f: f})
}

func (m *FieldSet) PreparedDispatch(ctx context.Context) bool {
	if len(m.delayed) == 0 {
		return true
	}

	fctx := GetFieldContext(ctx)
	//prepare
	if !fctx.IsPrepared {
		//one level preparing at a time
		parent := fctx
		for {
			if parent == nil {
				break
			}
			if parent.IsPreparing {
				return false //stop
			}
			parent = parent.Parent
		}

		fctx.IsPreparing = true
		fmt.Println(`Preparing values: ` + fctx.Object)
		for _, d := range m.delayed {
			m.Values[d.i] = d.f()
		}
		fctx.IsPreparing = false
		fctx.IsPrepared = true

		//update master prepare count
		parent = fctx
		for {
			if parent == nil {
				break
			}
			if parent.MasterPrepare {
				parent.MasterPrepareCount++
				break //stop
			}
			parent = parent.Parent
		}
		return false
	}

	fmt.Println(`Getting values: ` + fctx.Object)
	//fetch
	for _, d := range m.delayed {
		m.Values[d.i] = d.f()
	}
	return true
}

func (m *FieldSet) Dispatch(ctx context.Context) {
	if len(m.delayed) == 1 {
		// only one concurrent task, no need to spawn a goroutine or deal create waitgroups
		d := m.delayed[0]
		m.Values[d.i] = d.f()
	} else if len(m.delayed) > 1 {
		// more than one concurrent task, use the main goroutine to do one, only spawn goroutines for the others

		var wg sync.WaitGroup
		for _, d := range m.delayed[1:] {
			wg.Add(1)
			//go func(d delayedResult) {
			m.Values[d.i] = d.f()
			wg.Done()
			//}(d)
		}

		m.Values[m.delayed[0].i] = m.delayed[0].f()
		wg.Wait()
	}
}

func (m *FieldSet) MarshalGQL(writer io.Writer) {
	writer.Write(openBrace)
	for i, field := range m.fields {
		if i != 0 {
			writer.Write(comma)
		}
		writeQuotedString(writer, field.Alias)
		writer.Write(colon)
		if m.Values[i] == nil {
			writer.Write(colon)
		}
		m.Values[i].MarshalGQL(writer)
	}
	writer.Write(closeBrace)
}
