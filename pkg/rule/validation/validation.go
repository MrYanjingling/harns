package validation

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/rule/runtime"
	v1 "lightiot/pkg/rule/v1"
	validateutil "lightiot/pkg/util/validation"
	"strings"
)

func ValidateRule(obj *v1.Rule) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, meta.Validate(obj.Name, validateRuleName)...)
	allErrs = append(allErrs, generic.ValidateDescription(obj.Description, field.NewPath("description"))...)
	allErrs = append(allErrs, validateEvaluations(obj.Evaluations, field.NewPath("evaluations"))...)
	return allErrs
}

func ValidateRuleUpdate(obj *v1.Rule, old *runtime.Rule) field.ErrorList {
	return nil
}

func validateRuleName(name string) error {
	if !validateutil.IsStringLengthInRange(name, 1, 64) {
		return fmt.Errorf("must have at most %d bytes", 64)
	}
	return nil
}

func validateEvaluations(evaluations []v1.Evaluation, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	if len(evaluations) == 0 {
		errs = append(errs, field.Required(path, ""))
	}

	expr, operands, err := Parse(evaluations[0].Expression)
	if err != nil {
		errs = append(errs, field.Invalid(path.Index(0).Child("expression"), evaluations[0].Expression, fmt.Sprintf("invalid expression [%s]", err.Error())))
	}
	klog.V(4).InfoS("Parsed expression", "expr", expr, "operands", operands)
	propertiesByPropertySet := map[string][]string{}
	for _, operand := range operands {
		res := strings.Split(operand, runtime.OperandUS)
		if len(res) != 2 {
			errs = append(errs, field.Invalid(path.Index(0).Child("expression"), evaluations[0].Expression, fmt.Sprintf("invalid operand [%s], it should be <propertySetName>\u001F<propertyName>", operand)))
			continue
		}
		ps := res[0]
		p := res[1]
		if properties, ok := propertiesByPropertySet[ps]; ok {
			properties = append(properties, p)
		} else {
			propertiesByPropertySet[ps] = []string{p}
		}
	}

	if len(propertiesByPropertySet) > 1 {
		klog.V(2).InfoS("Rule was only able to refer a property set")
		errs = append(errs, field.Invalid(path.Index(0).Child("expression"), evaluations[0].Expression, fmt.Sprintf("real time rule refers more than one property set")))
	}

	return errs
}
