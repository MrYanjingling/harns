package thing

import (
	"lightiot/pkg/model/runtime"
	"time"
)

func SetDefaults_ThingType(tt *runtime.ThingType) {
	if tt.Instantiable == nil {
		t := true
		tt.Instantiable = &t
	}
}

func SetDefaults_Thing(tt *runtime.Thing) {
	if tt.TimeZone == nil {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		tt.TimeZone = (*runtime.TimeZone)(loc)
	}
}
