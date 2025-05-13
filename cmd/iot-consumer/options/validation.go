package options

import (
	"fmt"
	"lightiot/pkg/consumer/job"
)

func Validate(o *Options) []error {
	var errs []error
	if o.MaxInstances < 2 {
		errs = append(errs, fmt.Errorf("--max-consumers %d must be more than 1", o.MaxInstances))
	}
	if o.HeartbeatInterval < 1000 {
		errs = append(errs, fmt.Errorf("--heartbeat-interval %d must be more than 1000 (1s)", o.HeartbeatInterval))
	}

	var invalidJobGroup []string
	for _, jg := range o.JobGroups {
		if _, ok := job.GroupFromString[jg]; !ok {
			invalidJobGroup = append(invalidJobGroup, jg)
		}
	}
	if len(invalidJobGroup) != 0 {
		errs = append(errs, fmt.Errorf("--job-groups %v are not valid, choose from: %v", invalidJobGroup, job.GetGroups()))
	}

	if o.RetrieveJobInterval < 4 {
		errs = append(errs, fmt.Errorf("--retrieve-job-interval %d must be great than 3 sec", o.RetrieveJobInterval))
	}

	if o.RetrieveJobInterval > 10*(o.HeartbeatInterval/1000) {
		errs = append(errs, fmt.Errorf("--retrieve-job-interval %d must be less than 10 times --heartbeat-interval %d", o.RetrieveJobInterval, o.HeartbeatInterval/1000))
	}

	if err := o.BaseOptions.ValidateAndApply(); err != nil {
		errs = append(errs, err)
	}

	return errs
}
