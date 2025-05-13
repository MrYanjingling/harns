package meta

import (
	"fmt"
	"time"
)

var (
	ErrNotObject = fmt.Errorf("object does not implement the Object interfaces")
)

type ObjectMetaAccessor interface {
	GetObjectMeta() Object
}

type Object interface {
	GetTenant() string
	SetTenant(string)
	GetName() string
	SetName(string)
	GetID() string
	SetID(string)
	GetVersion() string
	SetVersion(string)
	GetModTime() time.Time
	SetModTime(time.Time)
}

func (meta *ObjectMeta) GetTenant() string            { return meta.Tenant }
func (meta *ObjectMeta) SetTenant(tenant string)      { meta.Tenant = tenant }
func (meta *ObjectMeta) GetName() string              { return meta.Name }
func (meta *ObjectMeta) SetName(name string)          { meta.Name = name }
func (meta *ObjectMeta) GetID() string                { return meta.ID }
func (meta *ObjectMeta) SetID(id string)              { meta.ID = id }
func (meta *ObjectMeta) GetVersion() string           { return meta.Version }
func (meta *ObjectMeta) SetVersion(version string)    { meta.Version = version }
func (meta *ObjectMeta) GetModTime() time.Time        { return meta.ModTime }
func (meta *ObjectMeta) SetModTime(modTime time.Time) { meta.ModTime = modTime }

func Accessor(obj interface{}) (Object, error) {
	switch t := obj.(type) {
	case Object:
		return t, nil
	case ObjectMetaAccessor:
		if m := t.GetObjectMeta(); m != nil {
			return m, nil
		}
		return nil, ErrNotObject
	default:
		return nil, ErrNotObject
	}
}
