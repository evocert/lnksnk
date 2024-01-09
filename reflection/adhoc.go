package reflection

import (
	"reflect"
)

func ReflectCallMethod(sender interface{}, methodname string, args ...interface{}) (called bool, result []interface{}) {
	if sender != nil {
		val := reflect.ValueOf(sender)
		valtype := reflect.TypeOf(sender)
		//if val.Kind() == reflect.Pointer {
		//	val = val.Elem()
		//}
		if mthd, mthdfnd := valtype.MethodByName(methodname); mthdfnd {
			argsl := len(args)
			argsvls := make([]reflect.Value, argsl)
			for an, a := range args {
				argsvls[an] = reflect.ValueOf(a)
			}

			rflctresult := val.Method(mthd.Index).Call(argsvls)
			if resultvlsL := len(rflctresult); resultvlsL > 0 {
				result = make([]interface{}, resultvlsL)
				for resultn := range result {
					result[resultn] = ReflectValToVal(rflctresult[resultn])
				}
			}
			called = true
		}
	}
	return
}

func ReflectMethods(sender interface{}, mthds ...string) (called bool, methods []string) {
	if sender != nil {
		//val := reflect.ValueOf(sender)
		valtype := reflect.TypeOf(sender)
		//if val.Kind() == reflect.Pointer {
		//	val = val.Elem()
		//}
		if mthdnm := valtype.NumMethod(); mthdnm > 0 {
			for mthdi := 0; mthdi < mthdnm; mthdi++ {
				mthdfnd := valtype.Method(mthdi)
				methods = append(methods, mthdfnd.Name)
			}
			called = true
		}
	}
	return
}

func ReflectCallField(sender interface{}, fieldname string, args ...interface{}) (called bool, result []interface{}) {
	if sender != nil {
		val := reflect.ValueOf(sender)
		valtype := reflect.TypeOf(sender)
		if val.Kind() == reflect.Pointer {
			val = val.Elem()
			valtype = val.Type()
		}

		if fld, fldfnd := valtype.FieldByName(fieldname); fldfnd {
			argsl := len(args)

			argsvls := make([]reflect.Value, argsl)
			for an, a := range args {
				argsvls[an] = reflect.ValueOf(a)
			}
			fldval := val.FieldByIndex(fld.Index)
			if len(argsvls) == 1 {
				fldval.Set(argsvls[0])
			}
			result = make([]interface{}, 1)
			for resultn := range result {
				result[resultn] = ReflectValToVal(reflect.ValueOf(fldval.Interface()))
			}

			called = true
		}
	}
	return
}

func ReflectFields(sender interface{}, flds ...string) (called bool, fields []string) {
	if sender != nil {
		val := reflect.ValueOf(sender)
		valtype := reflect.TypeOf(sender)
		if val.Kind() == reflect.Pointer {
			val = val.Elem()
			valtype = val.Type()
		}

		if fldnm := valtype.NumField(); fldnm > 0 {
			for fldi := 0; fldi < fldnm; fldi++ {
				fldfnd := valtype.Field(fldi)
				fields = append(fields, fldfnd.Name)
			}
			called = true
		}
	}
	return
}

func ReflectValToVal(vl reflect.Value) (val interface{}) {
	switch vl.Kind() {
	case reflect.Bool:
		val = vl.Bool()
	case reflect.String:
		val = vl.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val = func() interface{} {
			switch vl.Kind() {
			case reflect.Int:
				return int(vl.Int())
			case reflect.Int8:
				return int8(vl.Int())
			case reflect.Int16:
				return int16(vl.Int())
			case reflect.Int32:
				return int32(vl.Int())
			case reflect.Int64:
				return vl.Int()
			default:
				return int(vl.Int())
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		val = func() interface{} {
			switch vl.Kind() {
			case reflect.Uint:
				return uint(vl.Uint())
			case reflect.Uint8:
				return uint8(vl.Uint())
			case reflect.Uint16:
				return uint16(vl.Uint())
			case reflect.Uint32:
				return uint32(vl.Uint())
			case reflect.Uint64:
				return vl.Uint()
			case reflect.Uintptr:
				return uintptr(vl.Uint())
			default:
				return uint(vl.Uint())
			}
		}
	case reflect.Float32, reflect.Float64:
		val = func() interface{} {
			switch vl.Kind() {
			case reflect.Float32:
				return float32(vl.Float())
			case reflect.Float64:
				return vl.Float()
			default:
				return vl.Float()
			}
		}()
	case reflect.Complex64, reflect.Complex128:
		val = func() interface{} {
			switch vl.Kind() {
			case reflect.Complex64:
				return complex64(vl.Complex())
			case reflect.Complex128:
				return vl.Complex()
			default:
				return vl.Complex()
			}
		}()
	//	Array
	//	Chan
	//	Func
	case reflect.Interface:
		val = vl.Interface()
		//Map
		//Pointer
		//Slice

		//Struct
	case reflect.UnsafePointer:
	}
	return
}
