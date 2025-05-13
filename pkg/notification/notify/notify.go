package notify

import "lightiot/pkg/notification/runtime"

type Notifier interface {
	Notify(stopCh <-chan struct{}, msg *runtime.Message) (bool, error)
}
