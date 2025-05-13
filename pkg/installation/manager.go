package installation

import (
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/runtime"
	v1 "lightiot/pkg/installation/v1"
	"lightiot/pkg/objectstorage"
	"lightiot/pkg/storage"
	"lightiot/pkg/util/randutil"
	"lightiot/pkg/util/security"
	"lightiot/pkg/util/uuidutil"
	"mime/multipart"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

type Manager struct {
	installations *sync.Map // id --> *Installation
	iCh           chan runtime.Object
	iResCh        chan *storage.PersistResult
	store         *generic.Store
	stopCh        <-chan struct{}
	objectStore   *objectstorage.FsClient

	myselfId string
}

func NewManager(stopCh <-chan struct{}, iCh chan runtime.Object) *Manager {
	oss := &objectstorage.FsClient{}
	oss.Init(objectstorage.StoreTypeInstall)
	store, _ := generic.NewStore(runtime.GroupResource{storage.StoreGroupToString[storage.StoreGroupInstall], storage.Installations}, &Installation{}, iCh)
	return &Manager{
		installations: &sync.Map{},
		iCh:           iCh,
		stopCh:        stopCh,
		store:         store,
		objectStore:   oss,
	}
}

func (m *Manager) Init(myselfUrl string) error {
	is, _ := m.store.LoadResource()
	for _, obj := range is {
		i, _ := obj.(*Installation)
		m.installations.Store(i.ID, i)
	}

	m.iResCh = m.store.Start(m.stopCh)

	return m.registerMyself(myselfUrl)
}

func (m *Manager) CreateInstallation(obj *v1.Installation) (*Installation, error) {
	if len(obj.ProviderId) == 0 {
		obj.ProviderId = security.GetTenant()
	}

	exist := false
	predicates := parseFilter(installFilter{
		ProviderId: obj.ProviderId,
		Name:       obj.Name,
	})
	m.installations.Range(func(key, value interface{}) bool {
		isMatch := true
		v := value.(*Installation)
		for _, p := range predicates {
			if !p(v) {
				isMatch = false
				break
			}
		}
		if isMatch {
			exist = true
			return false
		}
		return true
	})
	if exist {
		klog.V(3).InfoS("Installation already exists", "name", obj.Name, "provider", obj.ProviderId)
		return nil, response.ErrResourceExists(fmt.Sprintf("%s-%s", obj.Name, obj.ProviderId))
	}

	ri := &Installation{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    obj.Name,
			ID:      uuidutil.UUID(),
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now(),
		},
		DisplayName: obj.DisplayName,
		ProviderId:  obj.ProviderId,
		Version:     obj.Version,
		Type:        obj.Type,
		Start:       obj.Start,
		End:         obj.End,
	}

	if len(obj.Components) != 0 {
		components := sets.String{}
		ri.Components = make([]*Component, 0)
		for _, c := range obj.Components {
			if components.Has(c.Name) {
				return nil, response.ErrDuplicatedComponent(c.Name)
			}
			if _, err := url.Parse(c.Uri); err != nil {
				klog.V(4).InfoS("Failed to parse uri", "component", c.Uri, "err", err)
				return nil, response.ErrComponentUriInvalid(c.Name, c.Uri)
			}
			rc := &Component{
				Name:      c.Name,
				Uri:       c.Uri,
				Endpoints: make([]*Endpoint, len(c.Endpoints)),
			}
			for i, e := range c.Endpoints {
				rc.Endpoints[i] = &Endpoint{
					Path:   e.Path,
					Action: e.Action,
					OsBar:  e.OsBar,
				}
			}
			ri.Components = append(ri.Components, rc)
		}
	}

	m.iCh <- ri
	res := <-m.iResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}
	m.installations.Store(ri.ID, res.Saved)
	return ri, nil
}

func (m *Manager) listInstallations(filter installFilter) []*Installation {
	is := make([]*Installation, 0)

	if len(filter.ProviderId) == 0 {
		filter.ProviderId = security.GetTenant()
	}
	predicates := parseFilter(filter)

	// descend
	byModTime := func(i1, i2 *Installation) bool { return i1.ModTime.Before(i2.ModTime) }
	sorter := By(byModTime)

	m.installations.Range(func(key, value interface{}) bool {
		isMatch := true
		v := value.(*Installation)
		for _, p := range predicates {
			if !p(v) {
				isMatch = false
				break
			}
		}
		if isMatch && !isFundamentalInstallation(v.Name) {
			is = sorter.Insert(is, v)
		}
		return true
	})
	return is
}

func (m *Manager) getInstallationById(id string) (*Installation, error) {
	r, isExist := m.installations.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return r.(*Installation), nil
}

func (m *Manager) deleteInstallationById(id string) (*Installation, error) {
	if id == m.myselfId {
		return nil, os.ErrPermission
	}
	r, ok := m.installations.Load(id)
	if !ok {
		return nil, os.ErrNotExist
	}
	i, _ := r.(*Installation)
	m.iCh <- &Installation{ObjectMeta: meta.ObjectMeta{ID: id, Version: i.Version}}
	res := <-m.iResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}
	m.installations.Delete(i.ID)
	return i, nil
}

func (m *Manager) saveImage(id, name string, file *multipart.FileHeader) error {
	if _, err := m.getInstallationById(id); err != nil {
		klog.V(3).InfoS("Installation not found", "id", id)
		return response.ErrResourceNotFound(id)
	}
	dst, err := m.objectStore.Create(path.Join(objectstorage.Images, id, name))
	if err != nil {
		return err
	}
	defer dst.Close()

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = io.Copy(dst, src)
	return err
}

func (m *Manager) listImages(id string) []*Image {
	var images []*Image
	if files, err := m.objectStore.List(path.Join(objectstorage.Images, id)); err != nil {
		klog.V(3).InfoS("Failed to list images", "installation", id, "err", err)
	} else {
		for _, file := range files {
			images = append(images, &Image{
				Name: file,
			})
		}
	}
	return images
}

func (m *Manager) getImage(id, name string) string {
	res, err := m.objectStore.Get(path.Join(objectstorage.Images, id, name))
	if err != nil {
		klog.V(3).InfoS("Failed to get image", "installation", id, "name", name, "err", err)
	}
	return res
}

func (m *Manager) deleteImage(id, name string) error {
	return m.objectStore.Delete(path.Join(objectstorage.Images, id, name))
}

func (m *Manager) registerMyself(myselfUrl string) error {
	if installs := m.listInstallations(installFilter{
		ProviderId: "main",
		Name:       "installation",
	}); len(installs) != 0 {
		m.myselfId = installs[0].ID
		klog.V(1).InfoS("Skipped register myself")
		return nil
	}

	actions := []v1.EndpointAction{
		v1.EndpointActionPost,
		v1.EndpointActionGet,
		v1.EndpointActionPut,
		v1.EndpointActionDelete,
	}
	endpoints := make([]v1.Endpoint, len(actions))
	for i, a := range actions {
		endpoints[i] = v1.Endpoint{
			Path:   group,
			Action: a,
		}
	}
	myself := v1.Installation{
		Name:        "installation",
		DisplayName: "Installation Manager",
		ProviderId:  "main",
		Version:     "v1",
		Type:        v1.InstallationTypeAPI,
		Components: []*v1.Component{
			{
				Name:      "im",
				Uri:       myselfUrl,
				Endpoints: endpoints,
			},
		},
	}
	if install, err := m.CreateInstallation(&myself); err != nil {
		klog.ErrorS(err, "Failed to register myself")
		return err
	} else {
		m.myselfId = install.ID
	}
	klog.V(1).InfoS("Succeed in registering myself")
	return nil
}

func isFundamentalInstallation(name string) bool {
	if name == "launchpad" ||
		name == "help" {
		return true
	}
	return false
}
