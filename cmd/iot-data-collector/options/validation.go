package options

import "fmt"

func Validate(o *Options) []error {
	var errs []error
	if o.JobDelaySec < 0 {
		errs = append(errs, fmt.Errorf("--job-delay %d must be natural number (non-negative integer)", o.JobDelaySec))
	}
	if err := o.Client.Validate(); len(err) != 0 {
		errs = append(errs, err...)
	}
	if err := o.BaseOptions.ValidateAndApply(); err != nil {
		errs = append(errs, err)
	}
	return errs
}
