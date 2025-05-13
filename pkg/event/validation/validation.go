package validation

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"lightiot/pkg/event/runtime"
	v1 "lightiot/pkg/event/v1"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	validateutil "lightiot/pkg/util/validation"
	"strings"
)

const (
	_illegalIDChars    = "\u002E\u002F\u003F\u005C" // ./?\
	_illegalFieldChars = "\u002E\u002F\u005C\u002C" // ./\
)

var (
	_preservedColumns = sets.NewString(
		// https://docs.influxdata.com/flux/v0.x/spec/lexical-elements/#keywords
		"and", "import", "not", "return", "option", "test", "empty", "in", "or", "package", "builtin",
		// https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree/#table_engine-mergetree-ttl
		"ttl",
		"id", "typeid", "correlationid", "_time", "thingid", "etag")
)

var _ImmutableEventType = sets.NewString(runtime.BaseEventTypeId, runtime.StandardEventTypeId)

func isValidID(id string) bool {
	return validateutil.ExcludesAll(id, _illegalIDChars)
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

func ValidateEventType(et *v1.EventType) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, meta.Validate(et.Name, ValidateTypeName)...)
	allErrs = append(allErrs, ValidateTTL(et.TTL, field.NewPath("ttl"))...)
	allErrs = append(allErrs, ValidateFields(et.Fields, field.NewPath("fields"))...)
	return allErrs
}

func ValidateEventTypeUpdate(newEt *v1.EventType, oldEt *runtime.EventType) (allErrs field.ErrorList) {
	if _ImmutableEventType.Has(oldEt.ID) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("id"), oldEt.ID, fmt.Sprintf("immutable eventtype %s", _ImmutableEventType.UnsortedList())))
		return
	}
	allErrs = append(allErrs, validation.ValidateImmutableField(newEt.Id, oldEt.ID, field.NewPath("id"))...)
	allErrs = append(allErrs, validation.ValidateImmutableField(newEt.Name, oldEt.Name, field.NewPath("name"))...)
	allErrs = append(allErrs, validation.ValidateImmutableField(newEt.ParentId, oldEt.ParentId, field.NewPath("parentId"))...)
	allErrs = append(allErrs, ValidateFieldsUpdate(newEt.Fields, oldEt.Fields, field.NewPath("fields"))...)
	return
}

func ValidateTTL(ttl *int, fldPath *field.Path) (allErrs field.ErrorList) {
	if ttl != nil && (*ttl < 0 || *ttl > generic.MaxTTLDay) {
		allErrs = append(allErrs, field.Invalid(fldPath, ttl, fmt.Sprintf("ttl should not be less than 0 or greater than %d", generic.MaxTTLDay)))
	}
	return
}

func ValidateFields(fields []*v1.Field, fldPath *field.Path) (allErrs field.ErrorList) {
	var errs field.ErrorList
	fieldSet := sets.NewString()
	for i, f := range fields {
		if fieldSet.Has(f.Name) {
			errs = append(errs, field.Duplicate(fldPath.Index(i).Child("name"), f.Name))
		} else {
			fieldSet.Insert(f.Name)
			errs = append(errs, validateFieldName(f.Name, fldPath.Index(i).Child("name"))...)
			if (f.DataType == v1.DatatypeMap || f.DataType == v1.DatatypeEnum) && f.Values == nil {
				errs = append(errs, field.Required(fldPath.Index(i).Child("values"), fmt.Sprintf("required for datatype %s", f.DataType)))
			}
		}
	}

	return errs
}

func ValidateFieldsUpdate(newFields []*v1.Field, oldFields []*runtime.Field, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if len(newFields) < len(oldFields) {
		allErrs = append(allErrs, field.Invalid(fldPath, newFields, "may not remove fields"))
		return allErrs
	}
	var removedFields []string
	for _, of := range oldFields {
		j := 0
		for range newFields {
			if newFields[j].Name == of.Name {
				nf := newFields[j]
				allErrs = append(allErrs, validation.ValidateImmutableField(nf.Filterable, of.Filterable, fldPath.Index(j).Child("filterable"))...)
				allErrs = append(allErrs, validation.ValidateImmutableField(nf.DataType, of.DataType, fldPath.Index(j).Child("datatype"))...)
				break
			}
			j++
		}
		if j == len(newFields) {
			removedFields = append(removedFields, of.Name)
		}
	}
	if len(removedFields) != 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, newFields, fmt.Sprintf("may not remove fields %s", removedFields)))
	}
	return allErrs
}

func isValidField(property string) bool {
	return validateutil.ExcludesAll(property, _illegalFieldChars)
}

func validateFieldName(name string, fldPath *field.Path) (errs field.ErrorList) {
	if len(name) == 0 {
		errs = append(errs, field.Required(fldPath, "required"))
		return
	}
	if !isValidField(name) {
		errs = append(errs, field.Invalid(fldPath, name, fmt.Sprintf("contains illegal chars [%s]", _illegalFieldChars)))
		return
	}
	if _preservedColumns.Has(strings.ToLower(name)) {
		errs = append(errs, field.Invalid(fldPath, name, fmt.Sprintf("contains preserved fields %s", _preservedColumns.UnsortedList())))
		return
	}
	if !validateutil.IsColumnNameLengthInRange(name, 1, 255-4) {
		errs = append(errs, field.TooLong(fldPath, name, 255-4))
	}
	return
}
