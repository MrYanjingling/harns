package options

func Validate(o *Options) []error {
	var errs []error
	if err := o.Client.Validate(); err != nil {
		errs = append(errs, err...)
	}
	if err := o.BaseOptions.ValidateAndApply(); err != nil {
		errs = append(errs, err)
	}
	return errs
}
