package v1

import (
	"encoding/json"
	"fmt"
)

func (ct ChannelType) MarshalJSON() ([]byte, error) {
	if s, ok := ChannelTypeToString[ct]; ok {
		return json.Marshal(s)
	}
	return nil, fmt.Errorf("unknown data type %d", ct)
}

func (ct *ChannelType) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	v, ok := ChannelTypeFromString[s]
	if !ok {
		return fmt.Errorf("unknown data type %s", s)
	}
	*ct = v
	return nil
}

func (ct ContentType) MarshalJSON() ([]byte, error) {
	if s, ok := ContentTypeToString[ct]; ok {
		return json.Marshal(s)
	}
	return nil, fmt.Errorf("unknown data type %d", ct)
}

func (ct *ContentType) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	v, ok := ContentTypeFromString[s]
	if !ok {
		return fmt.Errorf("unknown data type %s", s)
	}
	*ct = v
	return nil
}

func (s Secret) MarshalJSON() ([]byte, error) {
	if len(s) > 0 {
		return json.Marshal(SecretPlaceholder)
	} else {
		return nil, nil
	}
}