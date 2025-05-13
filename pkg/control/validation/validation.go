package validation

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"lightiot/pkg/control/runtime"
	v1 "lightiot/pkg/control/v1"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	validateutil "lightiot/pkg/util/validation"
	"strings"
)

const (
	_illegalIDChars     = "\u002E\u002F\u003F\u005C" // ./?\
	_illegalOptionChars = "\u002E\u002F\u005C"       // ./\
)

var (
	_preservedColumns = sets.NewString(
		// https://docs.influxdata.com/flux/v0.x/spec/lexical-elements/#keywords
		"and", "import", "not", "return", "option", "test", "empty", "in", "or", "package", "builtin",
		// https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree/#table_engine-mergetree-ttl
		"ttl",
		"seq", "thingid", "_time", "state", "code", "message",
		"typeid",
		"psn")
)

func ValidateCommandType(ct *v1.CommandType) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, meta.Validate(ct.Name, ValidateTypeName)...)
	allErrs = append(allErrs, generic.ValidateDescription(ct.Description, field.NewPath("description"))...)
	allErrs = append(allErrs, validateOptions(ct.Options, field.NewPath("options"))...)
	return allErrs
}

func ValidateCommandTypeUpdate(ct *v1.CommandType, old *runtime.CommandType) field.ErrorList {
	allErrs := validation.ValidateImmutableField(ct.Name, old.Name, field.NewPath("name"))
	allErrs = append(allErrs, validateInheritOptionUpdate(ct.Options, old.Options, old.OptionByName, field.NewPath("options"))...)
	allErrs = append(allErrs, validateOptionUpdate(ct.Options, old.Options, field.NewPath("options"))...)
	return allErrs
}

func validateInheritOptionUpdate(newOptions []*v1.Option, oldOptions []*runtime.Option, InheritOption map[string]*runtime.Option, path *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	ops := sets.NewString()
	for _, option := range oldOptions {
		ops.Insert(option.Name)
	}

	for i, option := range newOptions {
		if _, ok := InheritOption[option.Name]; ok && !ops.Has(option.Name) {
			allErrs = append(allErrs, field.Forbidden(path.Index(i), fmt.Sprintf("forbidden changing inherit options %s", option.Name)))
		}
	}
	return allErrs
}

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

func isValidOption(option string) bool {
	return validateutil.ExcludesAll(option, _illegalOptionChars)
}

func validateOptions(options []*v1.Option, fldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	names := sets.String{}
	for i, o := range options {
		if names.Has(o.Name) {
			errs = append(errs, field.Duplicate(fldPath.Index(i).Child("name"), o.Name))
			continue
		}
		errs = append(errs, validateOptionName(o.Name, fldPath.Index(i).Child("name"))...)
		errs = append(errs, generic.ValidateDescription(o.Description, fldPath.Index(i).Child("description"))...)
		if o.Datatype == v1.DatatypeEnum && len(o.Values) == 0 {
			errs = append(errs, field.Required(fldPath.Index(i).Child("values"), fmt.Sprintf("required for datatype %s", o.Datatype)))
		}
		names.Insert(o.Name)
	}
	return errs
}

func validateOptionName(name string, fldPath *field.Path) (errs field.ErrorList) {
	if len(name) == 0 {
		errs = append(errs, field.Required(fldPath, "required"))
		return
	}
	if !isValidOption(name) {
		errs = append(errs, field.Invalid(fldPath, name, fmt.Sprintf("contains illegal chars [%s]", _illegalOptionChars)))
		return
	}
	if _preservedColumns.Has(strings.ToLower(name)) {
		errs = append(errs, field.Invalid(fldPath, name, fmt.Sprintf("contains preserved option %s", _preservedColumns.List())))
		return
	}
	if !validateutil.IsColumnNameLengthInRange(name, 1, 255-4) {
		errs = append(errs, field.TooLong(fldPath, name, 255-4))
	}
	return
}

func validateOptionUpdate(newOptions []*v1.Option, oldOptions []*runtime.Option, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if len(newOptions) < len(oldOptions) {
		allErrs = append(allErrs, field.Invalid(fldPath, newOptions, "may not remove options"))
		return allErrs
	}
	var removedOptions []string
	for _, oo := range oldOptions {
		j := 0
		for range newOptions {
			if newOptions[j].Name == oo.Name {
				no := newOptions[j]
				allErrs = append(allErrs, validation.ValidateImmutableField(no.Datatype, oo.Datatype, fldPath.Index(j).Child("datatype"))...)
				allErrs = append(allErrs, validation.ValidateImmutableField(no.Filterable, oo.Filterable, fldPath.Index(j).Child("filterable"))...)
				break
			}
			j++
		}
		if j == len(newOptions) {
			removedOptions = append(removedOptions, oo.Name)
		}
	}
	if len(removedOptions) != 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, newOptions, fmt.Sprintf("may not remove options %s", removedOptions)))
	}
	return allErrs
}
