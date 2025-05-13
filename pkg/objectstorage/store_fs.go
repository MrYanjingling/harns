package objectstorage

import (
	"k8s.io/klog/v2"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type FsClient struct {
	storePath string
}

func (fc *FsClient) Init(st StoreType) {
	_, err := os.Stat(storePath)
	if err != nil {
		klog.Fatalf("%s: %v", storePath, err)
	}

	var dirs []string
	switch st {
	case StoreTypeInstall:
		dirs = []string{
			Images,
		}
		fc.storePath = filepath.Join(storePath, "installation")
	default:
		klog.Fatalf("Unsupported store type %d", st)
	}

	for _, m := range dirs {
		p := path.Join(fc.storePath, m)

		_, err = os.Stat(p)
		if os.IsNotExist(err) {
			absPath, _ := filepath.Abs(p)
			klog.V(2).InfoS("Created", "path", absPath)
			if err = os.MkdirAll(p, 0711); err != nil {
				klog.Fatal(err)
			}
		} else if err != nil {
			klog.Fatal(err)
		}
	}
}

func (fc *FsClient) Create(dst string) (*os.File, error) {
	i := strings.LastIndexByte(dst, '/')
	dir := dst[:i]
	p := path.Join(fc.storePath, dir)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		if err = os.Mkdir(p, 0711); err != nil {
			klog.V(2).InfoS("Failed to create dir", "err", err)
			return nil, err
		}
	}
	return os.OpenFile(path.Join(fc.storePath, dst), os.O_CREATE|os.O_RDWR, 0640)
}

func (fc *FsClient) List(dst string) ([]string, error) {
	var files []string
	if _, err := os.Stat(path.Join(fc.storePath, dst)); err != nil {
		return files, err
	}
	err := filepath.Walk(path.Join(fc.storePath, dst), func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, info.Name())
		}
		return nil
	})
	if err != nil {
		klog.V(2).InfoS("Failed to list", "err", err)
	}
	return files, nil
}

func (fc *FsClient) Get(dst string) (string, error) {
	if _, err := os.Stat(path.Join(fc.storePath, dst)); err != nil {
		return "", err
	} else {
		return path.Join(fc.storePath, dst), nil
	}
}

func (fc *FsClient) Delete(dst string) error {
	if _, err := os.Stat(path.Join(fc.storePath, dst)); err != nil {
		return err
	} else {
		return os.Remove(path.Join(fc.storePath, dst))
	}
}
