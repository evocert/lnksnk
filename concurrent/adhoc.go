package concurrent

import "reflect"

func constructValue(value interface{}, validkinds ...reflect.Kind) (result interface{}) {
	val, valok := value.(reflect.Value)
	if !valok {
		val = reflect.ValueOf(value)
	}
	if kind := val.Kind(); kind == reflect.Slice || kind == reflect.Array {
		vals := make([]reflect.Value, val.Len())
		values := make([]interface{}, val.Len())
		for n := range vals {
			values[n] = constructValue(vals[n])
		}
		valslice := NewSlize()
		valslice.Append(values...)
		result = valslice
	} else if kind == reflect.Map {
		keys := val.MapKeys()
		valmp := NewMap()
		for _, k := range keys {
			c_key := k.Convert(val.Type().Key())
			//c_value :=
			valmp.Set(constructValue(c_key), constructValue(val.MapIndex(c_key)))
		}
		result = valmp
	} else if kind == reflect.Invalid {
		result = nil
	} else {
		result = val.Interface()
	}
	return
}
