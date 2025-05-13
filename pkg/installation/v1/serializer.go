package v1

import (
	"encoding/json"
	"fmt"
)

const (
	EnabledOsBar  = "enabled"
	DisabledOsBar = "disabled"
)

func (it InstallationType) MarshalJSON() ([]byte, error) {
	if s, ok := InstallationTypeToString[it]; ok {
		return json.Marshal(s)
	}
	return nil, fmt.Errorf("unknown installation type %d", it)
}

func (it *InstallationType) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	v, ok := InstallationTypeFromString[s]
	if !ok {
		return fmt.Errorf("unknown installation type %s", s)
	}
	*it = v
	return nil
}

func (ea EndpointAction) MarshalJSON() ([]byte, error) {
	if s, ok := EndpointActionToString[ea]; ok {
		return json.Marshal(s)
	}
	return nil, fmt.Errorf("unknown endpoint action %d", ea)
}

func (ea *EndpointAction) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	v, ok := EndpointActionFromString[s]
	if !ok {
		return fmt.Errorf("unknown endpoint action %s", s)
	}
	*ea = v
	return nil
}

func (ob EnableOsBar) MarshalJSON() ([]byte, error) {
	if ob {
		return json.Marshal(EnabledOsBar)
	} else {
		return json.Marshal(DisabledOsBar)
	}
}

func (ob *EnableOsBar) UnmarshalJSON(bytes []byte) error {
	var eos string
	if err := json.Unmarshal(bytes, &eos); err != nil {
		return err
	}
	if eos == EnabledOsBar {
		*ob = true
	} else if eos == DisabledOsBar {
		*ob = false
	}
	return nil
}
