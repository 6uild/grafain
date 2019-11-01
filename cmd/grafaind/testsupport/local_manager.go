package testsupport

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8runtime "sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

func LocalManager() (manager.Manager, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	gv := schema.GroupVersion{Group: "", Version: "v1"}
	s, err := (&k8runtime.Builder{GroupVersion: gv}).Register(&corev1.Pod{}, &corev1.PodList{}).Build()
	if err != nil {
		return nil, err
	}
	opts := manager.Options{
		Scheme: s,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return FakeMapper{}, nil
		},
	}
	mgr, err := manager.New(cfg, opts)
	return mgr, err
}

type FakeMapper struct {
}

func (f FakeMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	panic("implement me")
}

func (f FakeMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	panic("implement me")
}

func (f FakeMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	panic("implement me")
}

func (f FakeMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	panic("implement me")
}

func (f FakeMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	panic("implement me")
}

func (f FakeMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	panic("implement me")
}

func (f FakeMapper) ResourceSingularizer(resource string) (singular string, err error) {
	panic("implement me")
}
