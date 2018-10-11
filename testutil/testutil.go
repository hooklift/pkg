package testutil

import (
	"flag"
	"log"
	"os"
	"testing"

	"github.com/c4milo/migrator"
	"github.com/hooklift/pkg/database"

	"github.com/golang/glog"
	"github.com/jmoiron/sqlx"
)

const (
	// UAADatabaseURL is the testing DB connection URL
	UAADatabaseURL = "postgres://hooklift@localhost/uaa_ci?sslmode=disable&application_name=uaa"
)

func init() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	flag.Parse()

	flag.Set("stderrthreshold", "FATAL")
	if testing.Verbose() {
		flag.Set("stderrthreshold", "ERROR")
		flag.Set("logtostderr", "true")
		flag.Set("v", "5")
	}
}

// TestMain runs sql scripts setting up the database for test scenarios.
func TestMain(m *testing.M, dbm database.Manager) {
	code := 1

	// deferred first so that the teardown function is called before.
	defer func() {
		os.Exit(code)
	}()

	if _, err := sqlx.LoadFile(dbm.Handle(), "./testdata/test-up.sql"); err != nil {
		glog.Warningf("Error loading SQL testdata: %v", err)
	}

	defer func() {
		if _, err := sqlx.LoadFile(dbm.Handle(), "./testdata/test-down.sql"); err != nil {
			glog.Warningf("Error unloading SQL testdata: %v", err)
		}
	}()

	glog.Info("Running tests...")
	// Run tests
	code = m.Run()
}

// DBManager returns a database connnection manager for use in testing. Takes a DB connection string as parameter.
func DBManager(url string, af migrator.AssetFunc,
	adf migrator.AssetDirFunc) database.Manager {
	dbm, err := database.GetManager(url, af, adf)
	if err != nil {
		panic(err)
	}

	return dbm
}
