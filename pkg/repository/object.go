package repository

import (
	"database/sql/driver"
	"encoding/json"
)

type Record interface {
	Get(string) (any, bool)
	Set(string, any)
	DeepCopy() Record
}

type Object map[string]any

var _ Record = (Object)(nil)

func (o Object) Get(key string) (any, bool) {
	if val, ok := o[key]; ok {
		return val, true
	}
	return nil, false
}

func (o Object) Set(key string, val any) {
	o[key] = val
}

func (o Object) DeepCopy() Record {
	// actually shadow copy
	copyMap := make(Object)
	for k, v := range o {
		copyMap[k] = v
	}
	return copyMap
}

func (o Object) Value() (driver.Value, error) {
	if o == nil {
		return nil, nil
	}
	return json.Marshal(o)
}

func (o *Object) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]uint8)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, o)
}

type Array []any

func (a Array) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

func (a *Array) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]uint8)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, a)
}
