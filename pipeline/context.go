package pipeline

import (
	"fmt"
	"reflect"
)

type ExecutionContext struct {
	values        map[reflect.Type][]reflect.Value
	initialValues []reflect.Value
}

func NewExecutionContext() *ExecutionContext {
	// creates an empty ExecutionContext.
	return &ExecutionContext{
		values:        make(map[reflect.Type][]reflect.Value),
		initialValues: []reflect.Value{},
	}
}

func (ctx *ExecutionContext) AddInputs(inputs ...interface{}) {
	// stores initial inputs in the context.
	for _, in := range inputs {
		val := reflect.ValueOf(in)
		t := val.Type()
		ctx.values[t] = append(ctx.values[t], val)
		ctx.initialValues = append(ctx.initialValues, val)
	}
}

func (ctx *ExecutionContext) StoreResults(results []reflect.Value) {
	// adds new result values to the context.
	for _, result := range results {
		ctx.storeValue(result)
	}
}

func (ctx *ExecutionContext) Values() map[reflect.Type][]reflect.Value {
	// returns all stored values keyed by type.
	return ctx.values
}

func (ctx *ExecutionContext) InitialValues() []reflect.Value {
	//  returns the initial input values.
	return ctx.initialValues
}

func (ctx *ExecutionContext) getValueByIndex(t reflect.Type, index int) (reflect.Value, error) {
	// retrieves a value of type t at the specified index.
	vals, ok := ctx.values[t]
	if !ok || len(vals) == 0 {
		return reflect.Value{}, fmt.Errorf("no values found for type %s", t)
	}
	if index < 0 {
		return reflect.Value{}, fmt.Errorf("invalid index (negative) for type %s", t)
	}
	if index >= len(vals) {
		index = len(vals) - 1 // clamp to last index
	}
	return vals[index], nil
}

func (ctx *ExecutionContext) storeValue(val reflect.Value) {
	// appends a new value to the context by its type.
	t := val.Type()
	ctx.values[t] = append(ctx.values[t], val)
}
