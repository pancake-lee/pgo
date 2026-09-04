package pdb

import (
	"path/filepath"
	"sync"
	"testing"

	glebarez "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type transactionTestRecord struct {
	ID   int `gorm:"primaryKey"`
	Name string
}

func newTransactionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(glebarez.Open(filepath.Join(t.TempDir(), "transaction.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&transactionTestRecord{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestNestedTransactionCommitsAfterEveryHandleFinishes(t *testing.T) {
	db := newTransactionTestDB(t)
	outer, err := BeginWithDB(101, db)
	if err != nil {
		t.Fatal(err)
	}
	if err := GetGormDB(101).Create(&transactionTestRecord{Name: "outer"}).Error; err != nil {
		t.Fatal(err)
	}
	inner, err := BeginWithDB(101, db)
	if err != nil {
		t.Fatal(err)
	}
	if err := GetGormDB(101).Create(&transactionTestRecord{Name: "inner"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := outer.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := inner.Commit(); err != nil {
		t.Fatal(err)
	}

	var count int64
	if err := db.Model(&transactionTestRecord{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func TestNestedTransactionRollsBackAfterEveryHandleFinishes(t *testing.T) {
	db := newTransactionTestDB(t)
	outer, err := BeginWithDB(102, db)
	if err != nil {
		t.Fatal(err)
	}
	if err := GetGormDB(102).Create(&transactionTestRecord{Name: "outer"}).Error; err != nil {
		t.Fatal(err)
	}
	inner, err := BeginWithDB(102, db)
	if err != nil {
		t.Fatal(err)
	}
	if err := GetGormDB(102).Create(&transactionTestRecord{Name: "inner"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := outer.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := inner.Rollback(); err != nil {
		t.Fatal(err)
	}

	var count int64
	if err := db.Model(&transactionTestRecord{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestTransactionRepeatedCloseAndDefaultLookup(t *testing.T) {
	db := newTransactionTestDB(t)
	tx, err := BeginWithDB(103, db)
	if err != nil {
		t.Fatal(err)
	}
	if GetGormDB(103) == db {
		t.Fatal("active transaction lookup returned base database")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("deferred rollback = %v", err)
	}
	if _, err := Begin(defaultTransactionID); err != ErrTransactionIDInvalid {
		t.Fatalf("err = %v", err)
	}
}

func TestTransactionRegistryIsConcurrentSafe(t *testing.T) {
	db := newTransactionTestDB(t)
	const transactionID int32 = 104
	const workerCount = 32
	transactionList := make(chan *Transaction, workerCount)
	var beginGroup sync.WaitGroup
	beginGroup.Add(workerCount)
	for range workerCount {
		go func() {
			defer beginGroup.Done()
			tx, err := BeginWithDB(transactionID, db)
			if err != nil {
				t.Errorf("BeginWithDB: %v", err)
				return
			}
			transactionList <- tx
		}()
	}
	beginGroup.Wait()
	close(transactionList)

	var finishGroup sync.WaitGroup
	for tx := range transactionList {
		finishGroup.Add(1)
		go func(tx *Transaction) {
			defer finishGroup.Done()
			if err := tx.Commit(); err != nil {
				t.Errorf("Commit: %v", err)
			}
		}(tx)
	}
	finishGroup.Wait()
	if GetGormDB(transactionID) != gDB {
		t.Fatal("completed transaction remained registered")
	}
}
