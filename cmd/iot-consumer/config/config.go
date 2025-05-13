package config

import (
	"lightiot/pkg/consumer/instance"
	"lightiot/pkg/consumer/job"
)

type Config struct {
	JobGroups        []job.Group
	InstanceConfig   instance.Config
}
