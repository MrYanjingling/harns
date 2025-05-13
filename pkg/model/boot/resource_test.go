package boot

import (
	"golang.org/x/mod/semver"
	"strings"
	"testing"
)

func TestEmbedFs(t *testing.T) {
	path := "thingtypes"
	entries, err := _resources.ReadDir(path)
	if err != nil {
		t.Log(err)
	}
	var vs []string
	for _, entry := range entries {
		t.Log(entry.Name())
		v := strings.Split(entry.Name(), "-")
		vs = append(vs, v[0])
	}

	semver.Sort(vs)

	t.Log(vs)
}
