package data

import (
	"lightiot/pkg/consumer/common"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/logstorage"
)

type Handler struct {
	logStore logstorage.Interface
	jm       *job.Manager
}

var _ common.DeleteHandler = (*Handler)(nil)

func NewHandler(jm *job.Manager, logStore logstorage.Interface) common.DeleteHandler {
	return &Handler{
		jm:       jm,
		logStore: logStore,
	}
}
