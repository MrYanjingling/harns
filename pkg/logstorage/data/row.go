package data

type Row struct {
	values map[string]interface{}
}

func NewRow(values map[string]interface{}) *Row {
	m := make(map[string]interface{}, 0)
	for k, v := range values {
		m[k] = v
	}
	return &Row{values: m}
}

func (r *Row) SetValue(k string, v interface{}) *Row {
	if r.values == nil {
		r.values = make(map[string]interface{}, 0)
	}
	r.values[k] = v
	return r
}

func (r *Row) GetValues() map[string]interface{} {
	return r.values
}