package runtime

import "path"

type GroupVersion struct {
	Group   string
	Version string
}

func (gv GroupVersion) String() string {
	return path.Join(gv.Group, gv.Version)
}

func (gv GroupVersion) WithResource(r string) GroupVersionResource {
	return GroupVersionResource{GroupVersion: gv, Resource: r}
}

type GroupResource struct {
	Group    string
	Resource string
}

type GroupVersionResource struct {
	GroupVersion
	Resource string
}

func (gvr GroupVersionResource) GroupResource() GroupResource {
	return GroupResource{Group: gvr.Group, Resource: gvr.Resource}
}
