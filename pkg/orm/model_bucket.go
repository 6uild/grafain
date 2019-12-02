package orm

import (
	"reflect"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	weaveORM "github.com/iov-one/weave/orm"
)

// Initial version copied from weave repo

// NewModelBucket returns a ModelBucket instance. This implementation relies on
// a bucket instance. Final implementation should operate directly on the
// KVStore instead.
func NewModelBucket(name string, m weaveORM.Model, opts ...ModelBucketOption) weaveORM.ModelBucket {
	b := weaveORM.NewBucket(name, m)

	tp := reflect.TypeOf(m)
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}

	mb := &modelBucket{
		b:     b,
		idSeq: b.Sequence("id"),
		model: tp,
	}
	for _, fn := range opts {
		fn(mb)
	}
	return mb
}

// ModelBucketOption is implemented by any function that can configure
// ModelBucket during creation.
type ModelBucketOption func(mb *modelBucket)

// WithIndex configures the bucket to build an index with given name. All
// entities stored in the bucket are indexed using value returned by the
// indexer function. If an index is unique, there can be only one entity
// referenced per index value.
func WithIndex(name string, indexer weaveORM.Indexer, unique bool) ModelBucketOption {
	return func(mb *modelBucket) {
		mb.b = mb.b.WithIndex(name, indexer, unique)
	}
}

// WithIndex configures the bucket to build an index with given name. All
// entities stored in the bucket are indexed using value returned by the
// indexer function. If an index is unique, there can be only one entity
// referenced per index value.
func WithMultiIndex(name string, indexer weaveORM.MultiKeyIndexer, unique bool) ModelBucketOption {
	return func(mb *modelBucket) {
		mb.b = mb.b.WithMultiKeyIndex(name, indexer, unique)
	}
}

// WithIDSequence configures the bucket to use the given sequence instance for
// generating ID.
func WithIDSequence(s weaveORM.Sequence) ModelBucketOption {
	return func(mb *modelBucket) {
		mb.idSeq = s
	}
}

type modelBucket struct {
	b     weaveORM.Bucket
	idSeq weaveORM.Sequence

	// model is referencing the structure type. Event if the structure
	// pointer is implementing Model interface, this variable references
	// the structure directly and not the structure's pointer type.
	model reflect.Type
}

func (mb *modelBucket) Register(name string, r weave.QueryRouter) {
	mb.b.Register(name, r)
}

func (mb *modelBucket) One(db weave.ReadOnlyKVStore, key []byte, dest weaveORM.Model) error {
	obj, err := mb.b.Get(db, key)
	if err != nil {
		return err
	}
	if obj == nil || obj.Value() == nil {
		return errors.Wrapf(errors.ErrNotFound, "%T not in the store", dest)
	}
	res := obj.Value()

	if !reflect.TypeOf(res).AssignableTo(reflect.TypeOf(dest)) {
		return errors.Wrapf(errors.ErrType, "%T cannot be represented as %T", res, dest)
	}

	reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(res).Elem())
	return nil
}

func (mb *modelBucket) ByIndex(db weave.ReadOnlyKVStore, indexName string, key []byte, destination weaveORM.ModelSlicePtr) ([][]byte, error) {
	objs, err := mb.b.GetIndexed(db, indexName, key)
	if err != nil {
		return nil, err
	}
	if len(objs) == 0 {
		return nil, nil
	}

	dest := reflect.ValueOf(destination)
	if dest.Kind() != reflect.Ptr {
		return nil, errors.Wrap(errors.ErrType, "destination must be a pointer to slice of models")
	}
	if dest.IsNil() {
		return nil, errors.Wrap(errors.ErrImmutable, "got nil pointer")
	}
	dest = dest.Elem()
	if dest.Kind() != reflect.Slice {
		return nil, errors.Wrap(errors.ErrType, "destination must be a pointer to slice of models")
	}

	// It is allowed to pass destination as both []MyModel and []*MyModel
	sliceOfPointers := dest.Type().Elem().Kind() == reflect.Ptr

	allowed := dest.Type().Elem()
	if sliceOfPointers {
		allowed = allowed.Elem()
	}
	if mb.model != allowed {
		return nil, errors.Wrapf(errors.ErrType, "this bucket operates on %s model and cannot return %s", mb.model, allowed)
	}

	keys := make([][]byte, 0, len(objs))
	for _, obj := range objs {
		if obj == nil || obj.Value() == nil {
			continue
		}
		val := reflect.ValueOf(obj.Value())
		if !sliceOfPointers {
			val = val.Elem()
		}
		dest.Set(reflect.Append(dest, val))
		keys = append(keys, obj.Key())
	}
	return keys, nil

}

func (mb *modelBucket) Put(db weave.KVStore, key []byte, m weaveORM.Model) ([]byte, error) {
	mTp := reflect.TypeOf(m)
	if mTp.Kind() != reflect.Ptr {
		return nil, errors.Wrap(errors.ErrType, "model destination must be a pointer")
	}
	if mb.model != mTp.Elem() {
		return nil, errors.Wrapf(errors.ErrType, "cannot store %T type in this bucket", m)
	}

	if err := m.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid model")
	}

	if len(key) == 0 {
		var err error
		key, err = mb.idSeq.NextVal(db)
		if err != nil {
			return nil, errors.Wrap(err, "ID sequence")
		}
	}

	obj := weaveORM.NewSimpleObj(key, m)
	if err := mb.b.Save(db, obj); err != nil {
		return nil, errors.Wrap(err, "cannot store in the database")
	}
	return key, nil
}

func (mb *modelBucket) Delete(db weave.KVStore, key []byte) error {
	if err := mb.Has(db, key); err != nil {
		return err
	}
	return mb.b.Delete(db, key)
}

func (mb *modelBucket) Has(db weave.KVStore, key []byte) error {
	if key == nil {
		// nil key is a special case that would cause the store API to panic.
		return errors.ErrNotFound
	}

	// As long as we rely on the Bucket implementation to access the
	// database, we must refine the key.
	key = mb.b.DBKey(key)

	ok, err := db.Has(key)
	if err != nil {
		return err
	}
	if !ok {
		return errors.ErrNotFound
	}
	return nil
}

var _ weaveORM.ModelBucket = (*modelBucket)(nil)
