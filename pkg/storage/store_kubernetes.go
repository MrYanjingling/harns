package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	rw "k8s.io/client-go/tools/watch"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TODO replace json encode with gob encode once update operation is supported

type KubeClient struct {
	clientSet        *kubernetes.Clientset
	namespace        string
	storeGroup       string
	ListItemsVersion map[string]string
}

type ListItems struct {
	Items []Item
}

type Item struct {
	Path    string
	Data    []byte
	ModTime time.Time
}

const (
	namespaceFilePath      = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	storeLabelKeyGroup     = "afflatus.harns.cn/group"
	storeLabelKeyResource  = "afflatus.harns.cn/resource"
	storeAnnotationKeyName = "afflatus.harns.cn/name"
	storeBinaryDataKey     = "data"
)

var (
	_ Storage = (*KubeClient)(nil)
)

// get name and resource from give string
func getFromKey(key string) (name, resource string, err error) {
	s := strings.Split(key, "/")
	if len(s) != 2 || len(s[0]) == 0 || len(s[1]) == 0 {
		return "", "", errors.Errorf("unsupport key: %s", key)
	}

	switch s[0] {
	case PropertySetTypes, ThingTypes, AgentTypes:
		m := md5.Sum([]byte(key))
		name = hex.EncodeToString(m[:])
	default:
		name = strings.Replace(key, "/", ".", -1)
	}

	return name, s[0], nil
}

func (kc *KubeClient) Init(st StoreGroup) {
	switch st {
	case StoreGroupModel:
		kc.storeGroup = "model"
	case StoreGroupEvent:
		kc.storeGroup = "event"
	case StoreGroupData:
		kc.storeGroup = "rule"
	case StoreGroupInstall:
		kc.storeGroup = "installation"
	default:
		klog.Fatalf("Unsupported store type %d", st)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalln(err)
	}

	kc.clientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalln(err)
	}

	namespace, err := os.ReadFile(namespaceFilePath)
	if err != nil {
		klog.Fatalln(err)
	}
	kc.namespace = strings.TrimSpace(string(namespace))
}

func (kc *KubeClient) Create(key string, obj interface{}) (interface{}, error) {
	name, resource, err := getFromKey(key)
	if err != nil {
		klog.V(2).InfoS("Failed to get key", "err", err)
		return nil, err
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(obj); err != nil {
		klog.V(2).InfoS("Failed to marshal", "err", err)
		return nil, err
	}

	configmap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: kc.namespace,
			Labels: map[string]string{
				storeLabelKeyGroup:    kc.storeGroup,
				storeLabelKeyResource: resource,
			},
			Annotations: map[string]string{
				storeAnnotationKeyName: key,
			},
			ManagedFields: []metav1.ManagedFieldsEntry{
				{
					FieldsType: "FieldsV1",
					FieldsV1:   &metav1.FieldsV1{},
				},
			},
		},
		BinaryData: map[string][]byte{
			storeBinaryDataKey: buf.Bytes(),
		},
	}
	_, err = kc.clientSet.CoreV1().ConfigMaps(kc.namespace).Create(context.TODO(), configmap, metav1.CreateOptions{})
	if err != nil {
		klog.V(2).InfoS("Failed to persist", "err", err)
		return nil, err
	}

	return obj, nil
}

func (kc *KubeClient) Get(key string) (interface{}, error) {
	name, _, err := getFromKey(key)
	if err != nil {
		klog.V(2).InfoS("Failed to get key", "err", err)
		return nil, err
	}

	configMap, err := kc.clientSet.CoreV1().ConfigMaps(kc.namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		klog.V(2).InfoS("Failed to get", "err", err)
		return nil, err
	}

	return configMap.BinaryData[storeBinaryDataKey], nil
}

func findManagedFields(accessor metav1.Object, fieldManager string, operation metav1.ManagedFieldsOperationType) (metav1.ManagedFieldsEntry, bool) {
	objManagedFields := accessor.GetManagedFields()
	for _, mf := range objManagedFields {
		if mf.Manager == fieldManager && mf.Operation == operation {
			return mf, true
		}
	}
	return metav1.ManagedFieldsEntry{}, false
}

func getConfigMapModTime(cm *v1.ConfigMap) time.Time {
	var modTime time.Time
	// fieldManager: https://github.com/kubernetes/client-go/blob/v0.23.0/rest/config.go#L501
	if fieldsEntry, ok := findManagedFields(cm, filepath.Base(os.Args[0]), metav1.ManagedFieldsOperationUpdate); ok {
		modTime = fieldsEntry.Time.Time
	}
	return modTime
}

func (kc *KubeClient) List(key string) (interface{}, error) {
	configMaps, err := kc.clientSet.CoreV1().ConfigMaps(kc.namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set{
			storeLabelKeyGroup:    kc.storeGroup,
			storeLabelKeyResource: key,
		}.AsSelector().String(),
	})
	if err != nil {
		klog.V(2).InfoS("Failed to list", "err", err)
		return nil, err
	}

	items := make([]Item, len(configMaps.Items))
	for i, item := range configMaps.Items {
		items[i].Path = item.Annotations[storeAnnotationKeyName]
		items[i].Data = item.BinaryData[storeBinaryDataKey]
		items[i].ModTime = getConfigMapModTime(&item)
	}

	kc.ListItemsVersion[key] = configMaps.ResourceVersion
	return &ListItems{items}, nil
}

func (kc *KubeClient) Update(key, version string, obj interface{}) (interface{}, error) {
	name, _, err := getFromKey(key)
	if err != nil {
		klog.V(2).InfoS("Failed to get key", "err", err)
		return nil, err
	}

	configMap, err := kc.clientSet.CoreV1().ConfigMaps(kc.namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		klog.V(2).InfoS("Failed to get", "err", err)
		return nil, err
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(obj); err != nil {
		klog.V(2).InfoS("Failed to marshal", "err", err)
		return nil, err
	}
	configMap.BinaryData[storeBinaryDataKey] = buf.Bytes()

	_, err = kc.clientSet.CoreV1().ConfigMaps(kc.namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		klog.V(2).InfoS("Failed to persist", "err", err)
		return nil, err
	}

	return obj, nil
}

func (kc *KubeClient) Delete(key, version string) (interface{}, error) {
	name, _, err := getFromKey(key)
	if err != nil {
		klog.V(2).InfoS("Failed to get key", "err", err)
		return nil, err
	}

	err = kc.clientSet.CoreV1().ConfigMaps(kc.namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		klog.V(2).InfoS("Failed to delete", "err", err)
	}

	return nil, err
}

func (kc *KubeClient) Watch(stopCh <-chan struct{}, key string, rev string) (chan *Event, error) {
	ch := make(chan *Event)

	group, resource, err := getFromKey(key)
	if err != nil {
		klog.V(2).InfoS("Failed to get key", "err", err)
		return nil, err
	}

	watcher, err := rw.NewRetryWatcher(kc.ListItemsVersion[key],
		cache.NewListWatchFromClient(
			kc.clientSet.CoreV1().RESTClient(),
			"configmaps",
			kc.namespace,
			fields.SelectorFromSet(fields.Set(labels.Set{
				storeLabelKeyGroup:    group,
				storeLabelKeyResource: resource,
			}))))
	if err != nil {
		klog.Fatal(err)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.ResultChan():
				if !ok {
					klog.InfoS("There is no event")
					break
				}

				if obj, ok := event.Object.(*v1.ConfigMap); ok {
					name := obj.Annotations[storeAnnotationKeyName]
					switch event.Type {
					case watch.Added:
						klog.V(5).InfoS("Created", "name", name)
						ch <- &Event{
							Type: Create,
							Data: name,
						}
					case watch.Modified:
						klog.V(5).InfoS("Updated", "name", name)
						ch <- &Event{
							Type: Update,
							Data: name,
						}
					case watch.Deleted:
						klog.V(5).InfoS("Deleted", "name", name)
						ch <- &Event{
							Type: Remove,
							Data: name,
						}
					}
				} else if watch.Error == event.Type {
					err := event.Object.(*metav1.Status)
					klog.ErrorS(errors.New(err.String()), "Watcher")
				}
			case <-stopCh:
				klog.V(2).InfoS("Stopped watch process")
				watcher.Stop()
				return
			}
		}
	}()

	return ch, nil
}
