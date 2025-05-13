package rule

import (
	"lightiot/pkg/generic"
	"lightiot/pkg/rule/runtime"
	"time"
)

func SetDefaults_BaseAction(ba *runtime.BaseAction) {
	if ba.Interval == nil {
		ba.Interval = &generic.Duration{Duration: 2 * time.Minute}
	}
}
