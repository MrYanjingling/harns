package runtime

import (
	"bytes"
	"encoding/json"
	"lightiot/pkg/generic/runtime"
	"strings"
)

func (in *ThingType) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := *in
	if in.Description != nil {
		d := *in.Description
		out.Description = &d
	}
	out.CharacteristicByName = make(map[string]*Characteristic, len(in.Characteristics))
	if in.Characteristics != nil {
		out.Characteristics = make([]*Characteristic, len(in.Characteristics))
		for i, c := range in.Characteristics {
			copied := *c
			out.Characteristics[i] = &copied
			out.CharacteristicByName[copied.Name] = &copied
		}
	}
	out.PropertySetByName = make(map[string]*PropertySet, len(in.PropertySets))
	if in.PropertySets != nil {
		out.PropertySets = make([]*PropertySet, len(in.PropertySets))
		for i, ps := range in.PropertySets {
			copied := ps.DeepCopy()
			out.PropertySets[i] = copied
			out.PropertySetByName[copied.Name] = copied
		}
	}
	return &out
}

func (in *PropertySet) DeepCopy() *PropertySet {
	if in == nil {
		return nil
	}
	out := *in
	// deep copy exclusive pst, shared pst should not be deep copied
	if !strings.Contains(in.PropertySetType.ID, ".") {
		if copied := in.PropertySetType.DeepCopyObject(); copied != nil {
			out.PropertySetType = copied.(*PropertySetType)
		}
	}
	return &out
}

func (in *Thing) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := *in
	if in.Description != nil {
		d := *in.Description
		out.Description = &d
	}
	if in.Characteristics != nil {
		out.Characteristics = make([]Value, len(in.Characteristics))
		copy(out.Characteristics, in.Characteristics)
	}
	return &out
}

func (in *Property) DeepCopy() *Property {
	if in == nil {
		return nil
	}

	out := *in

	if in.Min != nil {
		m := *in.Min
		out.Min = &m
	}

	if in.Max != nil {
		m := *in.Max
		out.Max = &m
	}

	return &out
}

func (in *PropertySetType) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := *in
	if in.Description != nil {
		d := *in.Description
		out.Description = &d
	}
	out.PropertyByName = make(map[string]*Property, len(in.Properties))
	if in.Properties != nil {
		out.Properties = make([]*Property, len(in.Properties))
		for i, p := range in.Properties {
			copied := p.DeepCopy()
			out.Properties[i] = copied
			out.PropertyByName[copied.Name] = copied
		}
	}

	return &out
}

func (in *AgentType) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := *in
	if in.Description != nil {
		d := *in.Description
		out.Description = &d
	}
	out.DataSources = []*DataSource{}
	out.DataSourceByName = make(map[string]*DataSource)
	return &out
}

func deepCopyCustomData(in *interface{}) (out *interface{}) {
	if in == nil {
		return
	}
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(in)
	_ = json.NewDecoder(&buf).Decode(out)
	return
}

func (in *DataSource) DeepCopy() *DataSource {
	if in == nil {
		return nil
	}
	out := *in
	if in.Description != nil {
		d := *in.Description
		out.Description = &d
	}
	out.DataPointById = make(map[string]*DataPoint, len(in.DataPoints))
	if in.DataPoints != nil {
		out.DataPoints = make([]*DataPoint, len(in.DataPoints))
		for i, dp := range in.DataPoints {
			copied := *dp
			copied.CustomData = deepCopyCustomData(dp.CustomData)
			out.DataPoints[i] = &copied
			out.DataPointById[copied.Id] = &copied
		}
	}
	return &out
}

func (in *Agent) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := *in
	if in.Description != nil {
		d := *in.Description
		out.Description = &d
	}
	if in.TypeId != nil {
		id := *in.TypeId
		out.TypeId = &id
	}
	out.DataSourceByName = make(map[string]*DataSource, len(in.DataSources))
	if in.DataSources != nil {
		out.DataSources = make([]*DataSource, len(in.DataSources))
		for i, ds := range in.DataSources {
			copied := ds.DeepCopy()
			copied.CustomData = deepCopyCustomData(ds.CustomData)
			out.DataSources[i] = copied
			out.DataSourceByName[copied.Name] = copied
		}
	}
	return &out
}

func (in *DataPointMapping) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := *in

	return &out
}

func (in *ResponseModel) DeepCopyObject() runtime.Object {
	return in
}
