package pdb

import (
	"errors"
	"fmt"
	"sync"

	"gorm.io/gorm"
)

// ErrTransactionIDInvalid means the caller used the reserved default ID.
var ErrTransactionIDInvalid = errors.New("pdb: transaction ID -1 is reserved")

const defaultTransactionID int32 = -1

type transactionStatus uint8

const (
	transactionPending transactionStatus = iota
	transactionCommitted
	transactionRolledBack
)

// Transaction is one logical level in an ID-scoped transaction. Levels may be
// closed in any order. The database transaction completes only after every
// level has called Commit or Rollback.
type Transaction struct {
	transactionID int32
	session       *transactionSession
	status        transactionStatus
}

type transactionSession struct {
	db              *gorm.DB
	transactionList []*Transaction
}

// transactionRegistry protects both the registry and every session referenced
// by it. Begin, lookup, status changes, and final cleanup are atomic with
// respect to one another.
var transactionRegistry = struct {
	sync.Mutex
	byID map[int32]*transactionSession
}{byID: make(map[int32]*transactionSession)}

// Begin starts a transaction for transactionID, or joins an active one with
// the same ID. The ID is selected by business code and must be unique among
// unrelated concurrent workflows.
func Begin(transactionID int32) (*Transaction, error) {
	return begin(transactionID, gDB)
}

// BeginWithDB is equivalent to Begin but uses db for a newly created root
// transaction. It supports multi-database callers and isolated tests.
func BeginWithDB(transactionID int32, db *gorm.DB) (*Transaction, error) {
	return begin(transactionID, db)
}

func begin(transactionID int32, db *gorm.DB) (*Transaction, error) {
	if transactionID == defaultTransactionID {
		return nil, ErrTransactionIDInvalid
	}
	if db == nil {
		return nil, fmt.Errorf("pdb: database is not initialized")
	}

	transactionRegistry.Lock()
	defer transactionRegistry.Unlock()

	session := transactionRegistry.byID[transactionID]
	if session == nil {
		txDB := db.Begin()
		if txDB.Error != nil {
			return nil, txDB.Error
		}
		session = &transactionSession{db: txDB}
		transactionRegistry.byID[transactionID] = session
	}

	tx := &Transaction{transactionID: transactionID, session: session, status: transactionPending}
	session.transactionList = append(session.transactionList, tx)
	return tx, nil
}

// GetGormDB returns transactionID's root transaction when active; otherwise it
// returns the default database. The optional argument preserves GetGormDB() for
// current callers, while enabling transaction-aware callers to pass an ID.
func GetGormDB(transactionIDList ...int32) *gorm.DB {
	if len(transactionIDList) == 0 || transactionIDList[0] == defaultTransactionID {
		return gDB
	}

	transactionRegistry.Lock()
	defer transactionRegistry.Unlock()
	if session := transactionRegistry.byID[transactionIDList[0]]; session != nil {
		return session.db
	}
	return gDB
}

// Commit marks this logical level successful. It is safe to call repeatedly,
// including through a deferred Rollback after Commit.
func (tx *Transaction) Commit() error {
	return tx.finish(transactionCommitted)
}

// Rollback marks this logical level failed. Any rollback causes the root
// transaction to roll back once every logical level has finished.
func (tx *Transaction) Rollback() error {
	return tx.finish(transactionRolledBack)
}

func (tx *Transaction) finish(status transactionStatus) error {
	if tx == nil {
		return nil
	}

	transactionRegistry.Lock()
	defer transactionRegistry.Unlock()

	// A completed root transaction is commonly followed by deferred Rollback.
	if transactionRegistry.byID[tx.transactionID] != tx.session || tx.status != transactionPending {
		return nil
	}
	tx.status = status

	shouldFinish := true
	shouldRollback := false
	for _, transaction := range tx.session.transactionList {
		if transaction.status == transactionPending {
			shouldFinish = false
			break
		}
		if transaction.status == transactionRolledBack {
			shouldRollback = true
		}
	}
	if !shouldFinish {
		return nil
	}

	var err error
	if shouldRollback {
		err = tx.session.db.Rollback().Error
	} else {
		err = tx.session.db.Commit().Error
	}
	delete(transactionRegistry.byID, tx.transactionID)
	return err
}
