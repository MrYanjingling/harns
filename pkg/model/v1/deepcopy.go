package v1

import "lightiot/pkg/generic/runtime"

func (in *ThingType) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}

	out := *in

	if in.Characteristics != nil {
		out.Characteristics = make([]*Characteristic, len(in.Characteristics))
		for i, c := range in.Characteristics {
			// !!! NOTE !!!
			// The comment code cannot work. It fails in UT.
			// It is different with value slice.
			// out.Characteristics[i] = &(*in.Characteristics[i])
			oc := *c
			out.Characteristics[i] = &oc
		}
	}

	if in.PropertySets != nil {
		out.PropertySets = make([]*PropertySet, len(in.PropertySets))
		for i, ps := range in.PropertySets {
			ops := *ps
			// if ps.Properties != nil {
			// 	ops.Properties = make([]*Property, len(ps.Properties))
			// 	for j, p := range ps.Properties {
			// 		ops.Properties[j] = p.DeepCopy()
			// 	}
			// }
			out.PropertySets[i] = &ops
		}
	}

	return &out
}

func (in *Thing) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}

	out := *in

	if in.Characteristics != nil {
		out.Characteristics = make([]Value, len(in.Characteristics))
		// for i := range in.Characteristics {
		// 	out.Characteristics[i] = in.Characteristics[i]
		// }
		copy(out.Characteristics, in.Characteristics)
	}
	return &out
}

func (in *Property) DeepCopy() *Property {
	if in == nil {
		return nil
	}

	out := *in

	return &out
}

func (in *PropertySetType) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}

	out := *in

	if in.Properties != nil {
		out.Properties = make([]*Property, len(in.Properties))
		for i, p := range in.Properties {
			out.Properties[i] = p.DeepCopy()
		}
	}

	return &out
}

func (in *DataPoint) DeepCopy() *DataPoint {
	if in == nil {
		return nil
	}

	out := *in

	return &out
}

func (in *DataSource) DeepCopy() *DataSource {
	if in == nil {
		return nil
	}

	out := *in

	if in.DataPoints != nil {
		out.DataPoints = make([]*DataPoint, len(in.DataPoints))
		for i, dp := range in.DataPoints {
			out.DataPoints[i] = dp.DeepCopy()
		}
	}

	return &out
}

func (in *AgentType) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}

	out := *in

	if in.DataSources != nil {
		out.DataSources = make([]*DataSource, len(in.DataSources))
		for i, ds := range in.DataSources {
			out.DataSources[i] = ds.DeepCopy()
		}
	}

	return &out
}

func (in *Agent) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}

	out := *in

	if in.DataSources != nil {
		out.DataSources = make([]*DataSource, len(in.DataSources))
		for i, ds := range in.DataSources {
			out.DataSources[i] = ds.DeepCopy()
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
