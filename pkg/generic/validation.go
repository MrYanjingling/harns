package generic

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/util/validation"
)

type ValidateNameFunc func(name string) error

func ValidateName(name string, nameFn meta.ValidateNameFunc, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if len(name) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), ""))
	} else if err := nameFn(name); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), name, err.Error()))
	}
	return allErrs
}

func ValidateDescription(description *string, fldPath *field.Path) (allErrs field.ErrorList) {
	if description != nil && !validation.IsStringLengthInRange(*description, 0, 255) {
		allErrs = append(allErrs, field.TooLong(fldPath, description, 255))
	}
	return
}
