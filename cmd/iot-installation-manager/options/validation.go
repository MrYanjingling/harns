package options

import (
	"net/url"
)

func Validate(o *Options) []error {
	var errs []error
	if _, err := url.Parse(o.RegistryUrl); err != nil {
		errs = append(errs, err)
	}
	if err := o.BaseOptions.ValidateAndApply(); err != nil {
		errs = append(errs, err)
	}
	return errs
}
