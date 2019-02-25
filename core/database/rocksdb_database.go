/*
Author : lei.ji@okcoin.com
Time : 2018.10.10
*/

package database

import (
	"fmt"
	"os"
	"sync"

	logging "github.com/ok-chain/okchain/log"
	"github.com/tecbot/gorocksdb"
)

var logger = logging.MustGetLogger("RDBDatabase")

type RDBDatabase struct {
	fn string        // filename for reporting
	db *gorocksdb.DB // RocksDB instance

	quitLock sync.Mutex      // Mutex protecting the quit channel access
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database

	log *logging.Logger // Contextual logger tracking the database path
	ro  *gorocksdb.ReadOptions
	wo  *gorocksdb.WriteOptions
}

// NewLDBDatabase returns a LevelDB wrapped object.
func NewRDBDatabase(dirPath string, cache int, handles int) (*RDBDatabase, error) {
	logger := logging.MustGetLogger("RDBDatabase")

	if fi, err := os.Stat(dirPath); err == nil {
		if !fi.IsDir() {
			return nil, fmt.Errorf("rocksdb/storage: open %s: not a directory", dirPath)
		}
	} else if os.IsNotExist(err) || os.IsPermission(err) {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	// Ensure we have some minimal caching and file guarantees
	if cache < 16 {
		cache = 16
	}
	if handles < 16 {
		handles = 16
	}
	logger.Info("Allocated cache and file handles", "cache", cache, "handles", handles)

	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	db, err := gorocksdb.OpenDb(opts, dirPath)
	ro := gorocksdb.NewDefaultReadOptions()
	wo := gorocksdb.NewDefaultWriteOptions()
	if err != nil {
		return nil, err
	}
	return &RDBDatabase{
		fn:  dirPath,
		db:  db,
		log: logger,
		ro:  ro,
		wo:  wo,
	}, nil
}

// Path returns the path to the database directory.
func (db *RDBDatabase) Path() string {
	return db.fn
}

// Put puts the given key / value to the queue
func (db *RDBDatabase) Put(key []byte, value []byte) error {
	// Generate the data to write to disk, update the meter and write
	//value = rle.Compress(value)

	return db.db.Put(db.wo, key, value)
}

func (db *RDBDatabase) Has(key []byte) (bool, error) {
	_, err := db.db.Get(db.ro, key)
	if err != nil {
		return false, err
	}
	return true, err
}

// Get returns the given key if it's present.
func (db *RDBDatabase) Get(key []byte) ([]byte, error) {
	// Retrieve the key and increment the miss counter if not found
	dat, err := db.db.Get(db.ro, key)
	if err != nil {
		return nil, err
	}
	return dat.Data(), nil
}

// Delete deletes the key from the queue and database
func (db *RDBDatabase) Delete(key []byte) error {
	// Execute the actual operation
	return db.db.Delete(db.wo, key)
}

func (db *RDBDatabase) NewIterator() *gorocksdb.Iterator {
	ro := gorocksdb.NewDefaultReadOptions()
	ro.SetFillCache(false)
	it := db.db.NewIterator(ro)
	return it
}

//// NewIteratorWithPrefix returns a iterator to iterate over subset of database content with a particular prefix.
//func (db *RDBDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
//	return db.db.NewIterator(util.BytesPrefix(prefix), nil)
//}

func (db *RDBDatabase) Close() {
	// Stop the metrics collection to avoid internal database races
	db.quitLock.Lock()
	defer db.quitLock.Unlock()

	if db.quitChan != nil {
		errc := make(chan error)
		db.quitChan <- errc
		if err := <-errc; err != nil {
			db.log.Error("Metrics collection failed", "err", err)
		}
	}
	db.db.Close()
}

func (db *RDBDatabase) RDB() *gorocksdb.DB {
	return db.db
}

func (db *RDBDatabase) NewBatch() Batch {
	return &rdbBatch{db: db.db, b: gorocksdb.NewWriteBatch()}
}

type rdbBatch struct {
	db   *gorocksdb.DB
	b    *gorocksdb.WriteBatch
	size int
}

func (b *rdbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

func (b *rdbBatch) Write() error {
	wo := gorocksdb.NewDefaultWriteOptions()
	return b.db.Write(wo, b.b)
}

func (b *rdbBatch) ValueSize() int {
	return b.size
}

func (b *rdbBatch) Reset() {
	b.b.Clear()
	b.size = 0
}
