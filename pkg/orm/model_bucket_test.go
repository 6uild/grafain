package orm

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	weaveORM "github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestModelBucket(t *testing.T) {
	db := store.MemStore()

	b := NewModelBucket("cnts", &orm.Counter{})

	if _, err := b.Put(db, []byte("c1"), &orm.Counter{Count: 1}); err != nil {
		t.Fatalf("cannot save orm.Counter instance: %s", err)
	}

	var c1 orm.Counter
	if err := b.One(db, []byte("c1"), &c1); err != nil {
		t.Fatalf("cannot get c1 orm.Counter: %s", err)
	}
	if c1.Count != 1 {
		t.Fatalf("unexpected orm.Counter state: %d", c1)
	}

	if err := b.Delete(db, []byte("c1")); err != nil {
		t.Fatalf("cannot delete c1 orm.Counter: %s", err)
	}
	if err := b.Delete(db, []byte("unknown")); !errors.ErrNotFound.Is(err) {
		t.Fatalf("unexpected error when deleting unexisting instance: %s", err)
	}
	if err := b.One(db, []byte("c1"), &c1); !errors.ErrNotFound.Is(err) {
		t.Fatalf("unexpected error for an unknown model get: %s", err)
	}
}

func TestModelBucketPutSequence(t *testing.T) {
	db := store.MemStore()

	b := NewModelBucket("cnts", &orm.Counter{})

	// Using a nil key should cause the sequence ID to be used.
	key, err := b.Put(db, nil, &orm.Counter{Count: 111})
	if err != nil {
		t.Fatalf("cannot save orm.Counter instance: %s", err)
	}
	if !bytes.Equal(key, weavetest.SequenceID(1)) {
		t.Fatalf("first sequence key should be 1, instead got %d", key)
	}

	// Inserting an entity with a key provided must not modify the ID
	// generation orm.Counter.
	if _, err := b.Put(db, []byte("mycnt"), &orm.Counter{Count: 12345}); err != nil {
		t.Fatalf("cannot save orm.Counter instance: %s", err)
	}

	key, err = b.Put(db, nil, &orm.Counter{Count: 222})
	if err != nil {
		t.Fatalf("cannot save orm.Counter instance: %s", err)
	}
	if !bytes.Equal(key, weavetest.SequenceID(2)) {
		t.Fatalf("second sequence key should be 2, instead got %d", key)
	}

	var c1 orm.Counter
	if err := b.One(db, weavetest.SequenceID(1), &c1); err != nil {
		t.Fatalf("cannot get first orm.Counter: %s", err)
	}
	if c1.Count != 111 {
		t.Fatalf("unexpected orm.Counter state: %d", c1)
	}

	var c2 orm.Counter
	if err := b.One(db, weavetest.SequenceID(2), &c2); err != nil {
		t.Fatalf("cannot get first orm.Counter: %s", err)
	}
	if c2.Count != 222 {
		t.Fatalf("unexpected orm.Counter state: %d", c2)
	}
}

func TestModelBucketByIndex(t *testing.T) {
	cases := map[string]struct {
		IndexName  string
		QueryKey   string
		DestFn     func() weaveORM.ModelSlicePtr
		WantErr    *errors.Error
		WantResPtr []*orm.Counter
		WantRes    []orm.Counter
		WantKeys   [][]byte
	}{
		"find none": {
			IndexName:  "value",
			QueryKey:   "124089710947120",
			WantErr:    nil,
			WantResPtr: nil,
			WantRes:    nil,
			WantKeys:   nil,
		},
		"find one": {
			IndexName: "value",
			QueryKey:  "1",
			WantErr:   nil,
			WantResPtr: []*orm.Counter{
				{Count: 1001},
			},
			WantRes: []orm.Counter{
				{Count: 1001},
			},
			WantKeys: [][]byte{
				weavetest.SequenceID(1),
			},
		},
		"find two": {
			IndexName: "value",
			QueryKey:  "4",
			WantErr:   nil,
			WantResPtr: []*orm.Counter{
				{Count: 4001},
				{Count: 4002},
			},
			WantRes: []orm.Counter{
				{Count: 4001},
				{Count: 4002},
			},
			WantKeys: [][]byte{
				weavetest.SequenceID(3),
				weavetest.SequenceID(4),
			},
		},
		"non existing index name": {
			IndexName: "xyz",
			WantErr:   orm.ErrInvalidIndex,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			indexByBigValue := func(obj orm.Object) ([]byte, error) {
				c, ok := obj.Value().(*orm.Counter)
				if !ok {
					return nil, errors.Wrapf(errors.ErrType, "%T", obj.Value())
				}
				// Index by the value, ignoring anything below 1k.
				raw := strconv.FormatInt(c.Count/1000, 10)
				return []byte(raw), nil
			}
			b := NewModelBucket("cnts", &orm.Counter{}, WithIndex("value", indexByBigValue, false))

			if _, err := b.Put(db, nil, &orm.Counter{Count: 1001}); err != nil {
				t.Fatalf("cannot save orm.Counter instance: %s", err)
			}
			if _, err := b.Put(db, nil, &orm.Counter{Count: 2001}); err != nil {
				t.Fatalf("cannot save orm.Counter instance: %s", err)
			}
			if _, err := b.Put(db, nil, &orm.Counter{Count: 4001}); err != nil {
				t.Fatalf("cannot save orm.Counter instance: %s", err)
			}
			if _, err := b.Put(db, nil, &orm.Counter{Count: 4002}); err != nil {
				t.Fatalf("cannot save orm.Counter instance: %s", err)
			}

			var dest []orm.Counter
			keys, err := b.ByIndex(db, tc.IndexName, []byte(tc.QueryKey), &dest)
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
			assert.Equal(t, tc.WantKeys, keys)
			assert.Equal(t, tc.WantRes, dest)

			var destPtr []*orm.Counter
			keys, err = b.ByIndex(db, tc.IndexName, []byte(tc.QueryKey), &destPtr)
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
			assert.Equal(t, tc.WantKeys, keys)
			assert.Equal(t, tc.WantResPtr, destPtr)
		})
	}
}

func TestModelBucketPutWrongModelType(t *testing.T) {
	db := store.MemStore()
	b := NewModelBucket("cnts", &orm.Counter{})

	if _, err := b.Put(db, nil, &orm.MultiRef{Refs: [][]byte{[]byte("foo")}}); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to store wrong model type value: %s", err)
	}
}

func TestModelBucketOneWrongModelType(t *testing.T) {
	db := store.MemStore()
	b := NewModelBucket("cnts", &orm.Counter{})

	if _, err := b.Put(db, []byte("counter"), &orm.Counter{Count: 1}); err != nil {
		t.Fatalf("cannot save orm.Counter instance: %s", err)
	}

	var ref orm.MultiRef
	if err := b.One(db, []byte("counter"), &ref); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to get wrong model type value: %s", err)
	}
}

func TestModelBucketByIndexWrongModelType(t *testing.T) {
	db := store.MemStore()
	b := NewModelBucket("cnts", &orm.Counter{},
		WithIndex("x", func(o orm.Object) ([]byte, error) { return []byte("x"), nil }, false))

	if _, err := b.Put(db, []byte("counter"), &orm.Counter{Count: 1}); err != nil {
		t.Fatalf("cannot save orm.Counter instance: %s", err)
	}

	var refs []orm.MultiRef
	if _, err := b.ByIndex(db, "x", []byte("x"), &refs); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to find wrong model type value: %s: %v", err, refs)
	}

	var refsPtr []*orm.MultiRef
	if _, err := b.ByIndex(db, "x", []byte("x"), &refsPtr); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to find wrong model type value: %s: %v", err, refs)
	}

	var refsPtrPtr []**orm.MultiRef
	if _, err := b.ByIndex(db, "x", []byte("x"), &refsPtrPtr); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to find wrong model type value: %s: %v", err, refs)
	}
}

func TestModelBucketHas(t *testing.T) {
	db := store.MemStore()
	b := NewModelBucket("cnts", &orm.Counter{})

	if _, err := b.Put(db, []byte("counter"), &orm.Counter{Count: 1}); err != nil {
		t.Fatalf("cannot save orm.Counter instance: %s", err)
	}

	if err := b.Has(db, []byte("counter")); err != nil {
		t.Fatalf("an existing entity must return no error: %s", err)
	}

	if err := b.Has(db, nil); !errors.ErrNotFound.Is(err) {
		t.Fatalf("a nil key must return ErrNotFound: %s", err)
	}

	if err := b.Has(db, []byte("does-not-exist")); !errors.ErrNotFound.Is(err) {
		t.Fatalf("a non exists entity must return ErrNotFound: %s", err)
	}
}
