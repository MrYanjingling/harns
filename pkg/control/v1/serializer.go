package v1

import (
	"encoding/json"
	"fmt"
)

func (dt Datatype) MarshalJSON() ([]byte, error) {
	if s, ok := DatatypeToString[dt]; ok {
		return json.Marshal(s)
	}
	return nil, fmt.Errorf("unknown data type %d", dt)
}

func (dt *Datatype) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	v, ok := DatatypeFromString[s]
	if !ok {
		return fmt.Errorf("unknown data type %s", s)
	}
	*dt = v
	return nil
}
