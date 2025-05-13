package validation

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/model/runtime"
	v1 "lightiot/pkg/model/v1"
	validateutil "lightiot/pkg/util/validation"
	"strconv"
	"strings"
)

const (
	_illegalIDChars       = "\u002E\u002F\u003F\u005C" // ./?\
	_illegalPropertyChars = "\u002E\u002F\u005C\u002C" // ./\,
	_illegalNameChars     = "\u002F\u005C"             // /\
)

// https://docs.influxdata.com/flux/v0.x/spec/lexical-elements/#keywords
// https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree/#table_engine-mergetree-ttl
var (
	_preservedColumns = sets.NewString("and", "import", "not", "return", "option", "test", "empty", "in", "or", "package", "builtin", "ttl",
		"ps", "ti", "_time")
)

func isValidID(id string) bool {
	return validateutil.ExcludesAll(id, _illegalIDChars)
}

func isValidProperty(property string) bool {
	return validateutil.ExcludesAll(property, _illegalPropertyChars)
}

func isValidName(name string) bool {
	return validateutil.ExcludesAll(name, _illegalNameChars)
}

func ValidateTypeName(name string) error {
	if !isValidID(name) {
		return fmt.Errorf("contains illegal chars [%s]", _illegalIDChars)
	}
	// https://github.com/ClickHouse/ClickHouse/blob/v22.1.2.2-stable/src/Interpreters/DatabaseCatalog.cpp#L755-L761
	// 9 is our longest database name - "event-raw"
	// 12 is max tenant length
	length := 255 - 4 - 36 - 1 - 1 - 9 - 12 - 3
	if !validateutil.IsColumnNameLengthInRange(name, 1, length) {
		return fmt.Errorf("must have at most %d bytes", length)
	}
	return nil
}

func ValidateName(name string) error {
	if !isValidName(name) {
		return fmt.Errorf("contains illegal chars [%s]", _illegalNameChars)
	}
	if !validateutil.IsStringLengthInRange(name, 1, 64) {
		return fmt.Errorf("must have at most %d bytes", 64)
	}
	return nil
}

func ValidatePropertySetType(pst *v1.PropertySetType) field.ErrorList {
	allErrs := meta.Validate(pst.Name, ValidateTypeName)
	allErrs = append(allErrs, generic.ValidateDescription(pst.Description, field.NewPath("description"))...)
	allErrs = append(allErrs, validateProperties(pst.Properties, field.NewPath("properties"))...)
	return allErrs
}

func validatePropertyName(name string, fldPath *field.Path) (errs field.ErrorList) {
	if len(name) == 0 {
		errs = append(errs, field.Required(fldPath, "required"))
		return
	}
	if !isValidProperty(name) {
		errs = append(errs, field.Invalid(fldPath, name, fmt.Sprintf("contains illegal chars [%s]", _illegalPropertyChars)))
		return
	}
	if _preservedColumns.Has(strings.ToLower(name)) {
		errs = append(errs, field.Invalid(fldPath, name, fmt.Sprintf("contains preserved properties %s", _preservedColumns.List())))
		return
	}
	if !validateutil.IsColumnNameLengthInRange(name, 1, 255-4) {
		errs = append(errs, field.TooLong(fldPath, name, 255-4))
	}
	return
}

func validateProperties(properties []*v1.Property, fldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if len(properties) == 0 {
		errs = append(errs, field.Required(fldPath, ""))
	}
	names := sets.String{}
	for i, p := range properties {
		if names.Has(p.Name) {
			errs = append(errs, field.Duplicate(fldPath.Index(i).Child("name"), p.Name))
			continue
		}
		errs = append(errs, validatePropertyName(p.Name, fldPath.Index(i).Child("name"))...)
		names.Insert(p.Name)
	}
	return errs
}

func ValidatePropertySetTypeUpdate(newPst *v1.PropertySetType, oldPst *runtime.PropertySetType) field.ErrorList {
	allErrs := validation.ValidateImmutableField(newPst.Name, oldPst.Name, field.NewPath("name"))
	allErrs = append(allErrs, validatePropertyUpdate(newPst.Properties, oldPst.Properties, field.NewPath("properties"))...)
	return allErrs
}

func validatePropertyUpdate(newProperties []*v1.Property, oldProperties []*runtime.Property, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if len(newProperties) < len(oldProperties) {
		allErrs = append(allErrs, field.Invalid(fldPath, newProperties, "may not remove properties"))
		return allErrs
	}
	var removedProperties []string
	for _, op := range oldProperties {
		j := 0
		for range newProperties {
			if newProperties[j].Name == op.Name {
				np := newProperties[j]
				allErrs = append(allErrs, validation.ValidateImmutableField(np.Datatype, op.DataType, fldPath.Index(j).Child("unit"))...)
				allErrs = append(allErrs, validation.ValidateImmutableField(np.Unit, op.Unit, fldPath.Index(j).Child("datatype"))...)
				allErrs = append(allErrs, validation.ValidateImmutableField(np.Length, op.Length, fldPath.Index(j).Child("length"))...)
				if np.AccessMode != op.AccessMode && op.AccessMode == v1.AccessModeReadOnly {
					allErrs = append(allErrs, field.Forbidden(fldPath.Index(j).Child("accessMode"), "may not update accessMode from ReadOnly to Read&Write"))
				}
				break
			}
			j++
		}
		if j == len(newProperties) {
			removedProperties = append(removedProperties, op.Name)
		}
	}
	if len(removedProperties) != 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, newProperties, fmt.Sprintf("may not remove properties %s", removedProperties)))
	}
	return allErrs
}

func ValidateThingType(tt *v1.ThingType) field.ErrorList {
	allErrs := meta.Validate(tt.Name, ValidateTypeName)
	allErrs = append(allErrs, generic.ValidateDescription(tt.Description, field.NewPath("description"))...)
	allErrs = append(allErrs, validateCharacteristics(tt.Characteristics, field.NewPath("characteristics"))...)
	allErrs = append(allErrs, validatePropertySets(tt.PropertySets, field.NewPath("propertySets"))...)
	return allErrs
}

func validateCharacteristics(chars []*v1.Characteristic, fldPath *field.Path) (errs field.ErrorList) {
	for i, char := range chars {
		errs = append(errs, validatePropertyName(char.Name, fldPath.Index(i).Child("name"))...)
		errs = append(errs, ValidateCharacteristicValue(char.DataType, char.DefaultValue, char.Length, fldPath.Index(i).Child("defaultValue"))...)
	}
	return
}

func ValidateCharacteristicValue(datatype v1.Datatype, value string, length int, fldPath *field.Path) (errs field.ErrorList) {
	if len(value) == 0 {
		return
	}
	switch datatype {
	case v1.DataTypeString:
		if len(value) > length {
			errs = append(errs, field.TooLong(fldPath, value, length))
		}
	case v1.DataTypeInt, v1.DataTypeLong:
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			errs = append(errs, field.Invalid(fldPath, value, fmt.Sprintf("invalid %s", datatype)))
		}
	case v1.DataTypeDouble:
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			errs = append(errs, field.Invalid(fldPath, value, fmt.Sprintf("invalid %s", datatype)))
		}
	case v1.DataTypeBoolean:
		if _, err := strconv.ParseBool(value); err != nil {
			errs = append(errs, field.Invalid(fldPath, value, fmt.Sprintf("invalid %s", datatype)))
		}
	default:
		// should never be here
	}
	return
}

func validatePropertySets(pss []*v1.PropertySet, fldPath *field.Path) (errs field.ErrorList) {
	names := sets.String{}
	for i, ps := range pss {
		if names.Has(ps.Name) {
			errs = append(errs, field.Duplicate(fldPath.Index(i).Child("name"), ps.Name))
			continue
		}
		errs = append(errs, validatePropertyName(ps.Name, fldPath.Index(i).Child("name"))...)
		if ps.PropertySetTypeId == nil || len(*ps.PropertySetTypeId) == 0 {
			if ps.PropertySetType == nil {
				errs = append(errs, field.Required(fldPath.Index(i), "either propertySetTypeId or propertySetType is required"))
			} else {
				path := fldPath.Index(i).Child("propertySetType")
				errs = append(errs, generic.ValidateDescription(ps.PropertySetType.Description, path.Child("description"))...)
				errs = append(errs, validateProperties(ps.PropertySetType.Properties, path.Child("properties"))...)
			}
		}
		names.Insert(ps.Name)
	}
	return
}

func ValidateThingTypeUpdate(newTt *v1.ThingType, oldTt *runtime.ThingType) field.ErrorList {
	allErrs := validation.ValidateImmutableField(newTt.Name, oldTt.Name, field.NewPath("name"))
	allErrs = append(allErrs, validation.ValidateImmutableField(newTt.ParentTypeId, oldTt.ParentTypeId, field.NewPath("parentTypeId"))...)
	allErrs = append(allErrs, validateCharacteristicUpdate(newTt.Characteristics, oldTt.Characteristics, field.NewPath("characteristics"))...)
	allErrs = append(allErrs, validatePropertySetUpdate(newTt.PropertySets, oldTt.PropertySets, field.NewPath("propertySets"))...)
	return allErrs
}

func validateCharacteristicUpdate(newChars []*v1.Characteristic, oldChars []*runtime.Characteristic, fldPath *field.Path) (errs field.ErrorList) {
	for i, nc := range newChars {
		for _, oc := range oldChars {
			if nc.Name == oc.Name {
				errs = append(errs, validation.ValidateImmutableField(nc.DataType, oc.DataType, fldPath.Index(i).Child("datatype"))...)
				errs = append(errs, validation.ValidateImmutableField(nc.Searchable, oc.Searchable, fldPath.Index(i).Child("searchable"))...)
				if nc.DataType == oc.DataType && nc.DataType == v1.DataTypeString && nc.Length < oc.Length {
					errs = append(errs, field.Invalid(fldPath.Index(i).Child("length"), nc.Length, fmt.Sprintf("may not less %d", oc.Length)))
				}
			}
		}
	}
	return
}

func validatePropertySetUpdate(newPss []*v1.PropertySet, oldPss []*runtime.PropertySet, fldPath *field.Path) (errs field.ErrorList) {
	for i, nps := range newPss {
		for _, ops := range oldPss {
			if nps.Name == ops.Name {
				if (nps.PropertySetTypeId == nil || len(*nps.PropertySetTypeId) == 0) && nps.PropertySetType != nil {
					path := fldPath.Index(i).Child("propertySetType")
					errs = append(errs, validatePropertyUpdate(nps.PropertySetType.Properties, ops.PropertySetType.Properties, path.Child("properties"))...)
				}
			}
		}
	}
	return
}

func ValidateThing(t *v1.Thing) field.ErrorList {
	allErrs := meta.Validate(t.Name, ValidateName)
	allErrs = append(allErrs, generic.ValidateDescription(t.Description, field.NewPath("description"))...)
	if len(t.TypeId) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("typeId"), ""))
	}
	cfld := field.NewPath("characteristics")
	for i, c := range t.Characteristics {
		if len(c.Name) == 0 {
			allErrs = append(allErrs, field.Required(cfld.Index(i).Child("name"), ""))
		}
		if len(c.Value) == 0 {
			allErrs = append(allErrs, field.Required(cfld.Index(i).Child("value"), ""))
		}
	}
	return allErrs
}

func ValidateThingUpdate(newT *v1.Thing, oldT *runtime.Thing) field.ErrorList {
	allErrs := validation.ValidateImmutableField(newT.TypeId, oldT.Type.ID, field.NewPath("typeId"))
	if newT.ParentId == oldT.ID {
		allErrs = append(allErrs, field.Invalid(field.NewPath("parentId"), newT.ParentId, "parent may not point to yourself"))
	}
	return allErrs
}

func ValidateAgentType(at *v1.AgentType) field.ErrorList {
	allErrs := meta.Validate(at.Name, ValidateName)
	allErrs = append(allErrs, generic.ValidateDescription(at.Description, field.NewPath("description"))...)
	allErrs = append(allErrs, validateDataSources(at.DataSources, field.NewPath("dataSources"))...)
	return allErrs
}

func validateDataSources(dss []*v1.DataSource, fldPath *field.Path) (errs field.ErrorList) {
	names := sets.String{}
	for i, ds := range dss {
		if names.Has(ds.Name) {
			errs = append(errs, field.Duplicate(fldPath.Index(i).Child("name"), ds.Name))
			continue
		}
		errs = append(errs, generic.ValidateName(ds.Name, ValidateName, fldPath.Index(i))...)
		errs = append(errs, generic.ValidateDescription(ds.Description, fldPath.Index(i).Child("description"))...)
		errs = append(errs, validateDataPoints(ds.DataPoints, fldPath.Index(i).Child("dataPoints"))...)
		names.Insert(ds.Name)
	}
	return
}

func validateDataPoints(dps []*v1.DataPoint, fldPath *field.Path) (errs field.ErrorList) {
	ids := sets.String{}
	for i, dp := range dps {
		if ids.Has(dp.Id) {
			errs = append(errs, field.Duplicate(fldPath.Index(i).Child("id"), dp.Id))
			continue
		}
		if len(dp.Id) == 0 {
			errs = append(errs, field.Required(fldPath.Index(i).Child("id"), ""))
		}
		errs = append(errs, generic.ValidateDescription(dp.Description, fldPath.Index(i).Child("description"))...)
		if len(dp.Name) > 64 {
			errs = append(errs, field.TooLong(fldPath.Index(i).Child("name"), dp.Name, 64))
		}
		ids.Insert(dp.Id)
	}
	return
}

func ValidateAgentTypeUpdate(newTt *v1.AgentType, oldTt *runtime.AgentType) field.ErrorList {
	allErrs := validation.ValidateImmutableField(newTt.Name, oldTt.Name, field.NewPath("name"))
	return allErrs
}

func ValidateAgent(a *v1.Agent) field.ErrorList {
	allErrs := meta.Validate(a.Name, ValidateName)
	allErrs = append(allErrs, generic.ValidateDescription(a.Description, field.NewPath("description"))...)
	if len(a.ThingId) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("thingId"), "thingId"))
	}
	if a.TypeId == nil || len(*a.TypeId) == 0 {
		allErrs = append(allErrs, validateDataSources(a.DataSources, field.NewPath("dataSources"))...)
	}
	return allErrs
}

func ValidateAgentUpdate(newA *v1.Agent, oldA *runtime.Agent) field.ErrorList {
	allErrs := validation.ValidateImmutableField(newA.TypeId, oldA.TypeId, field.NewPath("typeId"))
	allErrs = append(allErrs, validation.ValidateImmutableField(newA.ThingId, oldA.ThingId, field.NewPath("thingId"))...)
	return allErrs
}
