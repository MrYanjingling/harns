package repository

import (
	"reflect"
)

// TODO remove reflect

func eq(x any, y any) bool {
	vx := reflect.ValueOf(x)
	vy := reflect.ValueOf(y)

	if vx.Type() != vy.Type() {
		if vx.Type().ConvertibleTo(vy.Type()) {
			vx = vx.Convert(vy.Type())
		} else if vy.Type().ConvertibleTo(vx.Type()) {
			vy = vy.Convert(vx.Type())
		} else {
			return false
		}
	}

	return vx.Interface() == vy.Interface()
}

func lt(x any, y any) bool {
	vx := reflect.ValueOf(x)
	vy := reflect.ValueOf(y)

	if vx.Type() != vy.Type() {
		if vx.Type().ConvertibleTo(vy.Type()) {
			vx = vx.Convert(vy.Type())
		} else if vy.Type().ConvertibleTo(vx.Type()) {
			vy = vy.Convert(vx.Type())
		} else {
			return false
		}
	}

	switch vx.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return vx.Int() < vy.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return vx.Uint() < vy.Uint()
	case reflect.Float32, reflect.Float64:
		return vx.Float() < vy.Float()
	case reflect.String:
		return vx.String() < vy.String()
	default:
		return false
	}
}

func lte(x any, y any) bool {
	vx := reflect.ValueOf(x)
	vy := reflect.ValueOf(y)

	if vx.Type() != vy.Type() {
		if vx.Type().ConvertibleTo(vy.Type()) {
			vx = vx.Convert(vy.Type())
		} else if vy.Type().ConvertibleTo(vx.Type()) {
			vy = vy.Convert(vx.Type())
		} else {
			return false
		}
	}

	switch vx.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return vx.Int() <= vy.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return vx.Uint() <= vy.Uint()
	case reflect.Float32, reflect.Float64:
		return vx.Float() <= vy.Float()
	case reflect.String:
		return vx.String() <= vy.String()
	default:
		return false
	}
}

func gt(x any, y any) bool {
	vx := reflect.ValueOf(x)
	vy := reflect.ValueOf(y)

	if vx.Type() != vy.Type() {
		if vx.Type().ConvertibleTo(vy.Type()) {
			vx = vx.Convert(vy.Type())
		} else if vy.Type().ConvertibleTo(vx.Type()) {
			vy = vy.Convert(vx.Type())
		} else {
			return false
		}
	}

	switch vx.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return vx.Int() > vy.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return vx.Uint() > vy.Uint()
	case reflect.Float32, reflect.Float64:
		return vx.Float() > vy.Float()
	case reflect.String:
		return vx.String() > vy.String()
	default:
		return false
	}
}

func gte(x any, y any) bool {
	vx := reflect.ValueOf(x)
	vy := reflect.ValueOf(y)

	if vx.Type() != vy.Type() {
		if vx.Type().ConvertibleTo(vy.Type()) {
			vx = vx.Convert(vy.Type())
		} else if vy.Type().ConvertibleTo(vx.Type()) {
			vy = vy.Convert(vx.Type())
		} else {
			return false
		}
	}

	switch vx.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return vx.Int() >= vy.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return vx.Uint() >= vy.Uint()
	case reflect.Float32, reflect.Float64:
		return vx.Float() >= vy.Float()
	case reflect.String:
		return vx.String() >= vy.String()
	default:
		return false
	}
}
