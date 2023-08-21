package storage

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var BoltBucket = []byte("nodes")
var StdCodec Codec = *NewCodec()
var ErrNotFound = errors.New("record not found")
var ErrBucketNotFound = errors.New("bucket not")

type Bolt struct {
	db *bolt.DB
}

// deletes all keys from a bucket
func (b *Bolt) NukeAllKeys() error {
	tx, err := b.db.Begin(true)
	if err != nil {
		return err
	}
	bb := tx.Bucket(BoltBucket)
	err = bb.ForEach((func(k, v []byte) error {
		err := bb.Delete(k)
		if err != nil {
			return err
		}
		return nil
	}))
	if err != nil {
		return err
	}
	// do we need this?
	err = tx.Commit()
	return err
}

func NewBolt(path string) (*Bolt, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if _, err = tx.CreateBucketIfNotExists(BoltBucket); err != nil {
			return fmt.Errorf("error creating bucket: %s with error: %w", BoltBucket, err)
		}

		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Bolt{db: db}, nil
}

func (b *Bolt) Modifier(key string, current []any, fn func(codec Codec, got []byte, current []any) ([]any, error)) error {
	tx, err := b.getTxn()
	if err != nil {
		return err
	}
	got, err := b.get(key, tx)
	if err != nil {
		return b.rollback(tx, err)
	}

	codec := *NewCodec()
	currentValue, err := fn(codec, got, current)
	if err != nil {
		return b.rollback(tx, err)
	}
	encodedToSet, err := codec.Encode(currentValue)
	if err != nil {
		return b.rollback(tx, err)
	}
	err = b.set(key, encodedToSet, tx)
	if err != nil {
		return b.rollback(tx, err)
	}
	return b.commit(tx)
}

func (b *Bolt) getTxn() (*bolt.Tx, error) {
	tx, err := b.db.Begin(true)
	if err != nil {
		return nil, err
	}
	return tx, err
}

func (b *Bolt) rollback(tx *bolt.Tx, err error) error {
	errRollback := tx.Rollback()
	if errRollback != nil {
		return fmt.Errorf("inner error: %s, %w", err.Error(), errRollback)
	}
	return err
}

func (b *Bolt) commit(tx *bolt.Tx) error {
	err := tx.Commit()
	if err != nil {
		err = tx.Rollback()
		return err
	}
	return nil
}

func (b *Bolt) set(key string, value []byte, tx *bolt.Tx) error {
	bucket := tx.Bucket(BoltBucket)
	err := bucket.Put([]byte(key), value)

	return err
}

func (b *Bolt) get(key string, tx *bolt.Tx) ([]byte, error) {
	var bz []byte

	bucket := tx.Bucket(BoltBucket)
	bz = bucket.Get([]byte(key))
	if bz == nil {
		return nil, ErrNotFound
	}

	return bz, nil
}

func (b *Bolt) Set(key string, value []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BoltBucket)
		return b.Put([]byte(key), value)
	})
}

func (b *Bolt) Get(key string) ([]byte, error) {
	var bz []byte

	err := b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BoltBucket)
		bz = b.Get([]byte(key))
		if bz == nil {
			return ErrNotFound
		}
		return nil
	})

	return bz, err
}

func (b *Bolt) Delete(key string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(BoltBucket).Delete([]byte(key))
	})
}

// Keys first collect a set of keys given a prefix using db.View
// When the keys are collected it runs fn on each key.
func (b *Bolt) Keys(prefix string, fn func(key string) (bool, error)) error {
	var keys []string
	p := []byte(prefix)
	err := b.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(BoltBucket).Cursor()
		for k, _ := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, _ = c.Next() {
			keys = append(keys, string(k))
		}
		return nil
	})
	if err != nil {
		return err
	}

	// At this point we have collected the keys, and the lock from db.View is
	// released.
	for _, key := range keys {
		ok, err := fn(string(key))
		if err != nil {
			return err
		}
		if !ok {
			break
		}
	}

	return nil
}

func (b *Bolt) Close() error {
	return b.db.Close()
}

func (b *Bolt) NumItems() (int, error) {
	numItemsInBucket := 0
	err := b.db.Update(func(tx *bolt.Tx) error {
		metricsBucket := tx.Bucket(BoltBucket)
		if metricsBucket == nil {
			return nil
		}
		numItemsInBucket = metricsBucket.Stats().KeyN
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("error getting number of items in bukcet: %w", err)
	}
	return numItemsInBucket, nil
}
