package testsupport

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//type fakeManager struct {
//	server *webhook.Server
//}
//
//func NewFakeManager(t *testing.T, port int) manager.Manager {
//	t.Helper()
//
//	svr := &webhook.Server{
//		Port: port,
//		Host: "127.0.0.1",
//	}
//	m := &fakeManager{
//		server: svr,
//	}
//	assert.Nil(t, m.SetFields(svr))
//	return m
//}
//
//func (f fakeManager) Add(manager.Runnable) error {
//	panic("implement me")
//}
//
//func (f *fakeManager) Start(c <-chan struct{}) error {
//	return f.server.Start(c)
//}
//
//func (f fakeManager) GetConfig() *rest.Config {
//	panic("implement me")
//}
//
//func (f fakeManager) GetScheme() *runtime.Scheme {
//	panic("implement me")
//}
//
//func (f fakeManager) GetClient() client.Client {
//	panic("implement me")
//}
//
//func (f fakeManager) GetFieldIndexer() client.FieldIndexer {
//	panic("implement me")
//}
//
//func (f fakeManager) GetCache() cache.Cache {
//	panic("implement me")
//}
//
//func (f fakeManager) GetEventRecorderFor(name string) record.EventRecorder {
//	panic("implement me")
//}
//
//func (f fakeManager) GetRESTMapper() meta.RESTMapper {
//	panic("implement me")
//}
//
//func (f fakeManager) GetAPIReader() client.Reader {
//	return &FakeReader{}
//}
//
//func (f fakeManager) GetWebhookServer() *webhook.Server {
//	return f.server
//}
//func (f *fakeManager) SetFields(i interface{}) error {
//	scheme := runtime.NewScheme()
//	v1.SchemeBuilder.AddToScheme(scheme)
//	v1beta1.AddToScheme(scheme)
//	if _, err := inject.SchemeInto(scheme, i); err != nil {
//		return err
//	}
//	if _, err := inject.InjectorInto(f.SetFields, i); err != nil {
//		return err
//	}
//	if _, err := inject.MapperInto(FakeMapper{}, i); err != nil {
//		return err
//	}
//	return nil
//}
//
//type FakeReader struct {
//}
//
//func (f FakeReader) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
//	panic("implement me")
//}
//
//func (f FakeReader) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
//	panic("implement me")
//}

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
