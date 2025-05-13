package generic

import (
	"bytes"
	"encoding/gob"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/runtime"
	"lightiot/pkg/storage"
	"os"
	"path/filepath"
	"reflect"
)

type Store struct {
	resource     runtime.GroupResource
	resourceType reflect.Type
	objCh        chan runtime.Object
	client       storage.Storage
}

var _ storage.Storage = new(Store)

func NewStore(resource runtime.GroupResource, obj runtime.Object, objCh chan runtime.Object) (*Store, error) {
	s := &Store{
		resource:     resource,
		resourceType: getTypeOfResource(obj),
		objCh:        objCh,
	}

	_, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST")
	if ok {
		// in kubernetes runtime
		client := &storage.KubeClient{}
		client.Init(storage.StoreGroupFromString[resource.Group])
		s.client = client
	} else {
		client := &storage.FsClient{}
		client.Init(storage.StoreGroupFromString[resource.Group])
		s.client = client
	}

	return s, nil
}

func (s *Store) Create(key string, obj interface{}) (interface{}, error) {
	return s.client.Create(key, obj)
}

func (s *Store) Get(key string) (interface{}, error) {
	return s.client.Get(key)
}

func (s *Store) List(key string) (interface{}, error) {
	return s.client.List(key)
}

func (s *Store) Update(key, version string, obj interface{}) (interface{}, error) {
	return s.client.Update(key, version, obj)
}

func (s *Store) Delete(key, version string) (interface{}, error) {
	return s.client.Delete(key, version)
}

func (s *Store) Watch(stopCh <-chan struct{}, key string, rev string) (ch chan *storage.Event, err error) {
	ch, err = s.client.Watch(stopCh, s.resource.Resource, rev)
	go s.processEvent(stopCh, ch)
	return
}

func (s *Store) processEvent(stopCh <-chan struct{}, ch chan *storage.Event) {
	for {
		select {
		case e, _ := <-ch:
			obj := reflect.New(s.resourceType).Interface().(runtime.Object)
			switch e.Type {
			case storage.Create:
				data, err := s.client.Get(filepath.Join(s.resource.Resource, e.Data.(string)))
				if err == nil {
					_ = gob.NewDecoder(bytes.NewReader(data.([]byte))).Decode(obj)
					s.objCh <- obj
				}
			case storage.Update:
				data, err := s.client.Get(filepath.Join(s.resource.Resource, e.Data.(string)))
				if err == nil {
					_ = gob.NewDecoder(bytes.NewReader(data.([]byte))).Decode(obj)
					s.objCh <- obj
				}
			case storage.Remove:
				data, ok := e.Data.(string)
				_, id := filepath.Split(data)
				if ok {
					accessor, _ := meta.Accessor(obj)
					accessor.SetID(id)
					s.objCh <- obj
				}
			default:
				klog.V(1).InfoS("Unsupported operation", "operation", e.Type, "group", s.resource.Group, "resource", s.resource.Resource)
			}
		case <-stopCh:
			klog.V(2).InfoS("Stopped model watch event")
			return
		}
	}
}

func (s *Store) Start(stopCh <-chan struct{}) (resCh chan *storage.PersistResult) {
	resCh = make(chan *storage.PersistResult)
	go func() {
		for {
			var (
				saved interface{}
				err   error
				ret   storage.PersistResult
			)
			select {
			case object, ok := <-s.objCh:
				if !ok {
					klog.InfoS("There is no event")
					// TODO recover
					return
				}
				accessor, _ := meta.Accessor(object)
				key := filepath.Join(s.resource.Resource, accessor.GetID())
				if accessor.GetModTime().IsZero() {
					_, err = s.Delete(key, accessor.GetVersion())
				} else if _, err = s.Get(key); err != nil {
					if saved, err = s.Create(key, object); err == nil {
						ret.Saved = saved.(runtime.Object)
					}
				} else {
					if saved, err = s.Update(key, accessor.GetVersion(), object); err == nil {
						ret.Saved = saved.(runtime.Object)
					}
				}

				ret.Err = err
				resCh <- &ret
			case <-stopCh:
				klog.V(2).InfoS("Stopped store", "group", s.resource.Group, "resource", s.resource.Resource)
				return
			}
		}
	}()
	return
}

func (s *Store) LoadResource() ([]runtime.Object, error) {
	objs, err := s.List(s.resource.Resource)
	if err != nil {
		return nil, err
	}

	var ret []runtime.Object
	if files, ok := objs.([]*storage.FileInfo); ok {
		for _, file := range files {
			func() {
				obj := reflect.New(s.resourceType).Interface().(runtime.Object)
				f, err := os.Open(file.Path)
				defer f.Close()
				if err != nil {
					klog.V(2).InfoS("Failed to open", "file", file.Path, "resource", s.resource.Resource, "err", err)
					return
				}
				if err = gob.NewDecoder(f).Decode(obj); err != nil {
					klog.V(3).InfoS("Failed to unmarshal", "file", file.Path, "resource", s.resource.Resource, "err", err)
					return
				}
				accessor, _ := meta.Accessor(obj)
				accessor.SetModTime(file.ModTime)
				ret = append(ret, obj)
			}()
		}
	} else if items, ok := objs.(*storage.ListItems); ok {
		for _, item := range items.Items {
			obj := reflect.New(s.resourceType).Interface().(runtime.Object)
			if err = gob.NewDecoder(bytes.NewReader(item.Data)).Decode(obj); err != nil {
				klog.V(3).InfoS("Failed to unmarshal", "err", err)
				continue
			}
			accessor, _ := meta.Accessor(obj)
			accessor.SetModTime(item.ModTime)
			ret = append(ret, obj)
		}
	}

	return ret, nil
}

func getTypeOfResource(obj runtime.Object) reflect.Type {
	t := reflect.TypeOf(obj)
	if t.Kind() != reflect.Ptr {
		panic("All types must be pointers to structs.")
	}
	t = t.Elem()
	if t.Kind() != reflect.Struct {
		panic("All types must be pointers to structs.")
	}
	return t
}
