package repository

import (
	"database/sql/driver"
	"encoding/json"
)

type Object map[string]any

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
