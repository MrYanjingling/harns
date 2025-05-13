package boot

import (
	"embed"
	"encoding/json"
	"fmt"
	"golang.org/x/mod/semver"
	"k8s.io/klog/v2"
	"lightiot/pkg/model/thing"
	v1 "lightiot/pkg/model/v1"
	"os"
	"path"
	"strings"
)

const (
	_path               = "thingtypes"
	_versionThingTypeId = "main.version"
	_initVersion        = "v0.0.0"
)

var (
	//go:embed thingtypes/*.json
	_resources embed.FS
)

type version struct {
	propertySetTypes []*v1.PropertySetType
	thingTypes       []*v1.ThingType
	version          string
}

func InitModel(tm thing.Manager) error {
	versions, entryByVersion, err := loadVersion()
	if err != nil {
		return err
	}
	return updateModel(tm, versions, entryByVersion)
}

func loadVersion() ([]string, map[string]string, error) {
	entries, err := _resources.ReadDir(_path)
	if err != nil {
		return nil, nil, err
	}

	versions := make([]string, 0, len(entries))
	entryByVer := make(map[string]string, len(entries))
	for _, entry := range entries {
		n := strings.Split(entry.Name(), "-")
		if !semver.IsValid(n[0]) {
			return nil, nil, fmt.Errorf("invalid file name %s", entry.Name())
		}
		if _, ok := entryByVer[n[0]]; ok {
			return nil, nil, fmt.Errorf("duplicated version %s", n[0])
		}
		versions = append(versions, n[0])
		entryByVer[n[0]] = entry.Name()
	}
	semver.Sort(versions)

	return versions, entryByVer, nil
}

func updateModel(tm thing.Manager, versions []string, entryByVersion map[string]string) error {
	curVer := _initVersion
	verThingType, _ := tm.GetThingTypeById(_versionThingTypeId, false)
	if verThingType != nil && verThingType.CharacteristicByName["version"] != nil {
		curVer = verThingType.CharacteristicByName["version"].DefaultValue
	}
	for _, ver := range versions {
		if semver.Compare(ver, curVer) > 0 {
			f := func(ver string) error {
				var v version
				f, err := os.Open(path.Join(_path, entryByVersion[ver]))
				if err != nil {
					klog.V(2).InfoS("Failed to open", "err", err)
					return err
				}
				defer f.Close()
				if err = json.NewDecoder(f).Decode(&v); err != nil {
					klog.V(2).InfoS("Failed to unmarshal", "err", err)
					return err
				}
				v.version = ver
				// todo create or update model
				return nil
			}

			if err := f(ver); err != nil {
				return err
			}
			// todo update main.version
			curVer = ver
		}
	}
	return nil
}
