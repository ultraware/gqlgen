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

func (m *FieldSet) Prepare(ctx context.Context) {
	if len(m.delayed) == 0 {
		return
	}

	fmt.Println(`Preparing values`)
	fctx := GetFieldContext(ctx)

	fctx.prepareCount = 0
	for _, d := range m.delayed {
		fctx.DoPrepare = true
		m.Values[d.i] = d.f()
		if fctx.DoPrepare == false {
			fctx.prepareCount++
		}
	}

	fctx.DoPrepare = false
}

func (m *FieldSet) PrepareSub(ctx context.Context) bool {
	if len(m.delayed) == 0 {
		return true //ok
	}

	fctx := GetFieldContext(ctx)
	if fctx.DoSubPrepare {
		fctx.subPrepareCount++
		return false //stop
	}
	if fctx.Parent != nil && fctx.Parent.DoSubPrepare {
		fctx.Parent.subPrepareCount++
		return false //stop
	}
	if fctx.Parent != nil && fctx.Parent.Parent != nil && fctx.Parent.Parent.DoSubPrepare {
		fctx.Parent.Parent.subPrepareCount++
		return false //stop
	}

	fmt.Println(`Preparing sub values`)
	fctx.DoSubPrepare = true
	fctx.DoPrepare = false

	fctx.subPrepareCount = 0
	for _, d := range m.delayed {
		m.Values[d.i] = d.f()
	}

	fctx.DoSubPrepare = false
	return true //ok
}

func (m *FieldSet) Dispatch(ctx context.Context) {
	if len(m.delayed) > 0 {
		fmt.Println(`Getting values`)
	}

	if len(m.delayed) > 1 {
		fctx := GetFieldContext(ctx)
		if fctx != nil && fctx.prepareCount == len(m.delayed) {
			for _, d := range m.delayed[1:] {
				m.Values[d.i] = d.f()
			}
			m.Values[m.delayed[0].i] = m.delayed[0].f()
			return
		}
	}

	if len(m.delayed) == 1 {
		// only one concurrent task, no need to spawn a goroutine or deal create waitgroups
		d := m.delayed[0]
		m.Values[d.i] = d.f()
	} else if len(m.delayed) > 1 {
		// more than one concurrent task, use the main goroutine to do one, only spawn goroutines for the others

		var wg sync.WaitGroup
		for _, d := range m.delayed[1:] {
			wg.Add(1)
			go func(d delayedResult) {
				m.Values[d.i] = d.f()
				wg.Done()
			}(d)
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
		m.Values[i].MarshalGQL(writer)
	}
	writer.Write(closeBrace)
}
