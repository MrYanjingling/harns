package objectstorage

import (
	"k8s.io/klog/v2"
	"os/user"
	"path"
)

var (
	storePath = getStorePath()
)

func getStorePath() string {
	if u, err := user.Current(); err == nil {
		return path.Join(u.HomeDir, "edgeiot")
	} else {
		klog.ErrorS(err, "Failed to get home dir")
		return "./edgeiot"
	}
}
