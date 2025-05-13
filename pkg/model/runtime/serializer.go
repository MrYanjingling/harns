package runtime

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"lightiot/pkg/generic/meta"
	"strings"
	"time"
)

// the solution is from http://choly.ca/post/go-json-marshalling/
func (tt *ThingType) MarshalJSON() ([]byte, error) {
	type Alias ThingType

	var parentTypeId *string = nil
	if len(tt.ParentTypeId) > 0 {
		parentTypeId = &tt.ParentTypeId
	}
	var characteristics []*Characteristic = nil
	if len(tt.Characteristics) > 0 {
		characteristics = tt.Characteristics
	}
	var propertySets []*PropertySet = nil
	if len(tt.PropertySets) > 0 {
		propertySets = tt.PropertySets
	}
	return json.Marshal(&struct {
		ParentTypeId    *string           `json:"parentTypeId,omitempty"`
		Characteristics []*Characteristic `json:"characteristics,omitempty"`
		PropertySets    []*PropertySet    `json:"propertySets,omitempty"`
		*Alias
	}{
		ParentTypeId:    parentTypeId,
		Characteristics: characteristics,
		PropertySets:    propertySets,
		Alias:           (*Alias)(tt),
	})
}

func (t *Thing) MarshalJSON() ([]byte, error) {
	type Alias Thing

	var parent *string = nil
	if t.Parent != nil {
		parent = &t.Parent.ID
	}
	var characteristics []Value = nil
	if len(t.Characteristics) > 0 {
		characteristics = t.Characteristics
	}
	return json.Marshal(&struct {
		Type            string  `json:"typeId"`
		Parent          *string `json:"parentId,omitempty"`
		Characteristics []Value `json:"characteristics,omitempty"`
		*Alias
	}{
		Type:            t.Type.ID,
		Parent:          parent,
		Characteristics: characteristics,
		Alias:           (*Alias)(t),
	})
}

func (t *Thing) UnmarshalJSON(bytes []byte) error {
	type Alias Thing

	ta := &struct {
		Type   string  `json:"typeId"`
		Parent *string `json:"parentId,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}

	if err := json.Unmarshal(bytes, ta); err != nil {
		return err
	}

	t.Type = &ThingType{
		ObjectMeta: meta.ObjectMeta{ID: ta.Type},
	}

	if ta.Parent != nil && len(*ta.Parent) > 0 {
		t.Parent = &Thing{
			ObjectMeta: meta.ObjectMeta{ID: *ta.Parent},
		}
	}

	return nil
}

func (tz *TimeZone) MarshalJSON() ([]byte, error) {
	return json.Marshal((*time.Location)(tz).String())
}

func (tz *TimeZone) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	loc, err := time.LoadLocation(str)
	if err != nil {
		return err
	}

	*tz = *(*TimeZone)(loc)
	return nil
}

func (tz *TimeZone) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode((*time.Location)(tz).String())
	return buf.Bytes(), err
}

func (tz *TimeZone) UnmarshalBinary(data []byte) error {
	var str string
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&str); err != nil {
		return err
	}

	loc, err := time.LoadLocation(str)
	if err != nil {
		return err
	}

	*tz = *(*TimeZone)(loc)
	return nil
}

func (at *AgentType) MarshalJSON() ([]byte, error) {
	type Alias AgentType

	var dataSources []*DataSource = nil
	if len(at.DataSources) > 0 {
		dataSources = at.DataSources
	}

	return json.Marshal(&struct {
		DataSources []*DataSource `json:"dataSources,omitempty"`
		*Alias
	}{
		DataSources: dataSources,
		Alias:       (*Alias)(at),
	})
}

func (ps *PropertySet) MarshalBinary() ([]byte, error) {
	type Alias PropertySet

	var buf bytes.Buffer
	out := ps

	// the propertySetType is pre-defined and shared by this tenant
	if strings.Contains(ps.PropertySetType.ID, ".") {
		out = ps.DeepCopy()
		out.PropertySetType = &PropertySetType{
			ObjectMeta: meta.ObjectMeta{ID: ps.PropertySetType.ID},
		}
	}
	err := gob.NewEncoder(&buf).Encode((*Alias)(out))
	return buf.Bytes(), err
}

func (ps *PropertySet) UnmarshalBinary(data []byte) error {
	type Alias PropertySet

	var in Alias
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&in); err != nil {
		return err
	}
	*ps = *(*PropertySet)(&in)
	return nil
}
