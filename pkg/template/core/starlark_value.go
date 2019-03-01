package core

import (
	"fmt"

	"go.starlark.net/starlark"
)

type StarlarkValueToGoValueConversion interface {
	AsGoValue() interface{}
}

type StarlarkValue struct {
	val starlark.Value
}

func NewStarlarkValue(val starlark.Value) StarlarkValue {
	return StarlarkValue{val}
}

// TODO rename AsGoValue()
func (e StarlarkValue) AsInterface() interface{} {
	return e.asInterface(e.val)
}

func (e StarlarkValue) AsString() (string, error) {
	if typedVal, ok := e.val.(starlark.String); ok {
		return string(typedVal), nil
	}
	return "", fmt.Errorf("expected starlark.String, but was %T", e.val)
}

func (e StarlarkValue) AsBool() (bool, error) {
	if typedVal, ok := e.val.(starlark.Bool); ok {
		return bool(typedVal), nil
	}
	return false, fmt.Errorf("expected starlark.Bool, but was %T", e.val)
}

func (e StarlarkValue) AsInt64() (int64, error) {
	if typedVal, ok := e.val.(starlark.Int); ok {
		i1, ok := typedVal.Int64()
		if ok {
			return i1, nil
		}
		return 0, fmt.Errorf("expected int64 value")
	}
	return 0, fmt.Errorf("expected starlark.Int")
}

func (e StarlarkValue) asInterface(val starlark.Value) interface{} {
	if obj, ok := val.(StarlarkValueToGoValueConversion); ok {
		return obj.AsGoValue()
	}

	switch typedVal := val.(type) {
	case nil, starlark.NoneType:
		return nil // TODO is it nil or is it None

	case starlark.Bool:
		return bool(typedVal)

	case starlark.String:
		return string(typedVal)

	case starlark.Int:
		i1, ok := typedVal.Int64()
		if ok {
			return i1
		}
		i2, ok := typedVal.Uint64()
		if ok {
			return i2
		}
		panic("not sure how to get int") // TODO

	case starlark.Float:
		return float64(typedVal)

	case *starlark.Dict:
		return e.dictAsInterface(typedVal)

	case *StarlarkStruct:
		return e.structAsInterface(typedVal)

	case *starlark.List:
		return e.itearableAsInterface(typedVal)

	case starlark.Tuple:
		return e.itearableAsInterface(typedVal)

	case *starlark.Set:
		return e.itearableAsInterface(typedVal)

	default:
		panic(fmt.Sprintf("unknown type %T for conversion to go value", val))
	}
}

func (e StarlarkValue) dictAsInterface(val *starlark.Dict) interface{} {
	result := map[interface{}]interface{}{}
	for _, item := range val.Items() {
		if item.Len() != 2 {
			panic("dict item is not KV")
		}
		result[e.asInterface(item.Index(0))] = e.asInterface(item.Index(1))
	}
	return result
}

func (e StarlarkValue) structAsInterface(val *StarlarkStruct) interface{} {
	// TODO accessing privates
	result := map[interface{}]interface{}{}
	for k, v := range val.data {
		result[k] = e.asInterface(v)
	}
	return result
}

func (e StarlarkValue) itearableAsInterface(iterable starlark.Iterable) interface{} {
	iter := iterable.Iterate()
	defer iter.Done()

	var result []interface{}
	var x starlark.Value
	for iter.Next(&x) {
		result = append(result, e.asInterface(x))
	}
	return result
}
