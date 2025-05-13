package v1

import (
	"encoding/json"
	"fmt"
)

func (dt Datatype) MarshalJSON() ([]byte, error) {
	if s, ok := DataTypeToString[dt]; ok {
		return json.Marshal(s)
	}
	return nil, fmt.Errorf("unknown data type %d", dt)
}

func (dt *Datatype) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	v, ok := DataTypeFromString[s]
	if !ok {
		return fmt.Errorf("unknown data type %s", s)
	}
	*dt = v
	return nil
}

func (am AccessMode) MarshalJSON() ([]byte, error) {
	s := "r"
	if am == AccessModeReadWrite {
		s = "rw"
	}
	return json.Marshal(s)
}

func (am *AccessMode) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	if s == "rw" {
		*am = AccessModeReadWrite
	} else {
		*am = AccessModeReadOnly
	}
	return nil
}

func (adt AgentDataType) MarshalJSON() ([]byte, error) {
	if s, ok := AgentDataTypeToString[adt]; ok {
		return json.Marshal(s)
	}
	return nil, fmt.Errorf("unknown agent data type %d", adt)
}

func (adt *AgentDataType) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	v, ok := AgentDataTypeFromString[s]
	if !ok {
		return fmt.Errorf("unknown agent data type %s", s)
	}
	*adt = v
	return nil
}
