package runtime

import (
	"encoding/json"
	"fmt"
	"lightiot/pkg/generic/meta"
)

func (ste State) MarshalJSON() ([]byte, error) {
	if s, ok := stateToString[ste]; ok {
		return json.Marshal(s)
	}
	return nil, fmt.Errorf("unknown data type %d", ste)
}

func (ste *State) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	v, ok := stateFromString[s]
	if !ok {
		return fmt.Errorf("unknown data type %s", s)
	}
	*ste = v
	return nil
}

func (ct CommandType) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		meta.ObjectMeta
		Description *string   `json:"description,omitempty"`
		Ack         bool      `json:"ack"`
		ThingTypeId string    `json:"thingTypeId"`
		Options     []*Option `json:"options,omitempty"`
	}{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  ct.Tenant,
			Name:    ct.Name,
			ID:      ct.TypeId,
			Version: ct.Version,
			ModTime: ct.ModTime,
		},
		Description: ct.Description,
		Ack:         ct.Ack,
		ThingTypeId: ct.ThingTypeId,
		Options:     ct.Options,
	})
}
