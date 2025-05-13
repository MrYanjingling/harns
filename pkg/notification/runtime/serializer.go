package runtime

import (
	"bytes"
	"encoding/gob"
)

func (pt *ParsedTmpl) MarshalBinary() ([]byte, error) {
	type Alias ParsedTmpl
	var buf bytes.Buffer
	var out Alias
	err := gob.NewEncoder(&buf).Encode(out)
	return buf.Bytes(), err
}

func (pt *ParsedTmpl) UnmarshalBinary(data []byte) error {
	type Alias ParsedTmpl

	var in Alias
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&in); err != nil {
		return err
	}
	*pt = *(*ParsedTmpl)(&in)
	return nil
}
