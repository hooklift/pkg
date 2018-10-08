package database

import (
	"context"
	"database/sql"
	"sync"

	"github.com/c4milo/migrator"
	"github.com/golang/glog"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var (
	// ErrUnsupportedDB is returned when a specific database is not supported
	// by this system. The only database supported so far is Postgres.
	ErrUnsupportedDB = errors.New("unsupported-database")
)

// Manager defines an interface for implementing different database managers.
type Manager interface {
	Create() error
	//Destroy() error
	Migrate() error
	Handle() *sqlx.DB
}

// Singleton DB Manager
var (
	managerInstance Manager
	once            sync.Once
)

// GetManager creates and initializes the database required by this system.
func GetManager(url string, af migrator.AssetFunc,
	adf migrator.AssetDirFunc) (Manager, error) {
	var err error
	once.Do(func() {
		managerInstance = NewManager(url, af, adf)

		err = managerInstance.Create()
		if err != nil {
			return
		}

		err = managerInstance.Migrate()
		if err != nil {
			return
		}
	})

	if err != nil {
		return nil, err
	}

	return managerInstance, nil
}

type txKey struct{}

// BeginTx creates and returns a new database transaction.
func BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return managerInstance.Handle().BeginTxx(ctx, nil)
}

// NewTxContext returns a context containing transaction instance.
func NewTxContext(ctx context.Context, tx *sqlx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFromContext extracts the transaction instance from the given context.
func TxFromContext(ctx context.Context) (tx *sqlx.Tx, ok bool) {
	tx, ok = ctx.Value(txKey{}).(*sqlx.Tx)
	return
}

// ResolveTx commits or rolls back the transaction depending of whether or not err is defined.
func ResolveTx(tx *sqlx.Tx, err error) {
	if err != nil {
		glog.V(4).Infof("rolling back DB tx due to %+v", err)
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			glog.Errorf("%+v", errors.Wrapf(err, "failed rolling back transaction"))
		}
	} else {
		glog.V(4).Info("committing DB tx")
		if err := tx.Commit(); err != nil && err != sql.ErrTxDone {
			glog.Errorf("%+v", errors.Wrapf(err, "failed commiting transaction"))
		}
	}
}
