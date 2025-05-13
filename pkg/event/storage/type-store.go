package storage

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/event/runtime"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	model "lightiot/pkg/model/runtime"
	"lightiot/pkg/storage"
)

type TypeStore struct {
	etStore  *generic.Store
	tStore   *generic.Store
	aetStore *generic.Store
	aetCh    chan gruntime.Object
	aetResCh chan *storage.PersistResult
}

var _ storage.Storage = new(TypeStore)

func NewTypeStore(etCh chan gruntime.Object, tCh chan gruntime.Object, aetCh chan gruntime.Object) (*TypeStore, error) {
	etStore, _ := generic.NewStore(gruntime.GroupResource{Group: storage.StoreGroupToString[storage.StoreGroupEvent], Resource: storage.EventTypes}, &runtime.EventType{}, etCh)
	tStore, _ := generic.NewStore(gruntime.GroupResource{Group: storage.StoreGroupToString[storage.StoreGroupModel], Resource: storage.Things}, &model.Thing{}, tCh)
	aetStore, _ := generic.NewStore(gruntime.GroupResource{Group: storage.StoreGroupToString[storage.StoreGroupEvent], Resource: storage.Things}, &runtime.ActiveEventType{}, aetCh)

	ts := &TypeStore{
		etStore:  etStore,
		tStore:   tStore,
		aetStore: aetStore,
		aetCh:    aetCh,
	}
	return ts, nil
}

func (ts *TypeStore) Create(key string, obj interface{}) (interface{}, error) {
	_, _ = ts.etStore.Create(key, obj)
	return nil, nil
}

func (ts *TypeStore) Get(key string) (interface{}, error) {
	return ts.etStore.Get(key)
}

func (ts *TypeStore) List(key string) (interface{}, error) {
	objs, _ := ts.etStore.List(key)
	return objs, nil
}

func (ts *TypeStore) Update(key, version string, obj interface{}) (interface{}, error) {
	return ts.etStore.Update(key, version, obj)
}

func (ts *TypeStore) Delete(key, version string) (interface{}, error) {
	return ts.etStore.Delete(key, version)
}

func (ts *TypeStore) Watch(stopCh <-chan struct{}, key string, _ string) (chan *storage.Event, error) {
	switch key {
	case storage.EventTypes:
		_, _ = ts.etStore.Watch(stopCh, "", "")
	case storage.Things:
		_, _ = ts.tStore.Watch(stopCh, "", "")
	}
	return nil, nil
}

func (ts *TypeStore) LoadEventTypes() ([]*runtime.EventType, error) {
	objs, _ := ts.etStore.LoadResource()
	ets := make([]*runtime.EventType, len(objs))
	for i, obj := range objs {
		ets[i] = obj.(*runtime.EventType)
	}
	return ets, nil
}

func (ts *TypeStore) LoadThings() (sets.String, sets.String, map[string]sets.String, error) {
	things := sets.NewString()
	objs, _ := ts.tStore.LoadResource()
	for _, obj := range objs {
		things.Insert(obj.(*model.Thing).ID)
	}

	activeThings := sets.NewString()
	eventTypesByThing := make(map[string]sets.String)
	objs, _ = ts.aetStore.LoadResource()
	for _, aet := range objs {
		ets := aet.(*runtime.ActiveEventType).EventTypes
		thingId := aet.(*runtime.ActiveEventType).ThingId
		activeThings.Insert(thingId)
		eventTypesByThing[thingId] = ets
	}
	return things, activeThings, eventTypesByThing, nil
}

func (ts *TypeStore) PruneThings(things sets.String) error {
	for _, t := range things.UnsortedList() {
		klog.V(4).InfoS("Thing has been deleted from model manager", "thingId", t)
		aet := &runtime.ActiveEventType{ObjectMeta: meta.ObjectMeta{ID: t}}
		ts.aetCh <- aet
		res := <-ts.aetResCh
		if res.Err != nil {
			klog.V(3).InfoS("Failed to delete active thing", "thingId", t, "err", res.Err)
		}
	}
	return nil
}

func (ts *TypeStore) SaveActivateEventType(aet *runtime.ActiveEventType) error {
	ts.aetCh <- aet
	res := <-ts.aetResCh
	if res.Err != nil {
		return res.Err
	}
	return nil
}

func (ts *TypeStore) Start(stopCh <-chan struct{}) (etResCh chan *storage.PersistResult) {
	_, _ = ts.Watch(stopCh, storage.Things, "")
	ts.aetResCh = ts.aetStore.Start(stopCh)
	return ts.etStore.Start(stopCh)
}
