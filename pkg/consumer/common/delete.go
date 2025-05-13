package common

import "lightiot/pkg/consumer/job"

type DeleteHandler interface {
	Delete(j *job.Job) error
}
