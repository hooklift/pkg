// +build postgres

package database

import (
	"database/sql"
	"html"
	neturl "net/url"

	"github.com/c4milo/migrator"
	"github.com/golang/glog"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

var db *sqlx.DB

type postgresManager struct {
	url          string
	username     string
	password     string
	host         string
	dbname       string
	assetFunc    migrator.AssetFunc
	assetDirFunc migrator.AssetDirFunc
}

// NewManager implements `manager` interface for PostgresSQL, enabled through build flags
func NewManager(url string, af migrator.AssetFunc, adf migrator.AssetDirFunc) Manager {
	if url == "" {
		glog.Fatalln("Database URL is required")
	}

	u, err := neturl.Parse(url)
	if err != nil {
		glog.Fatalf("%+v", errors.Wrapf(err, "failed parsing DB URL: %q", url))
	}

	m := new(postgresManager)
	m.url = url
	m.username = u.User.Username()
	m.password, _ = u.User.Password()
	m.dbname = u.Path[1:]
	m.host = u.Host
	m.assetFunc = af
	m.assetDirFunc = adf
	return m
}

// TODO(c4milo): Assign a password to the database user.
func (p *postgresManager) createUser(db *sqlx.DB) error {
	glog.V(2).Infof("Checking if database user %q exists...", p.username)

	var userExists int
	db.Get(&userExists, "SELECT 1 FROM pg_user WHERE usename = $1", p.username)
	if userExists == 1 {
		return nil
	}

	glog.V(2).Infof("Creating database user %q...", p.username)
	// Postgres does not support prepared DDLs
	db.MustExec(`CREATE ROLE ` + pq.QuoteIdentifier(html.EscapeString(p.username)) +
		` SUPERUSER CREATEDB CREATEROLE INHERIT LOGIN`)

	return nil
}

func (p *postgresManager) createDatabase(db *sqlx.DB) error {
	glog.V(2).Infof("Checking if database %q exists...", p.dbname)
	var dbExists int
	db.Get(&dbExists, "SELECT 1 FROM pg_database WHERE datname = $1", p.dbname)
	if dbExists == 1 {
		return nil
	}

	glog.V(2).Infof("Creating database %q...", p.dbname)
	sanitizedDBname := pq.QuoteIdentifier(html.EscapeString(p.dbname))

	// Postgres does not support prepared DDLs
	db.MustExec(`CREATE DATABASE ` + sanitizedDBname +
		` WITH OWNER ` + pq.QuoteIdentifier(html.EscapeString(p.username)))

	// We want the database to use UTC by default and left it up to the
	// user facing apps to translate results to the user's specific timezone.
	db.MustExec(`ALTER DATABASE ` + sanitizedDBname +
		` SET timezone to 'UTC'`)
	return nil
}

// TODO(c4milo): Enable SSL by default once we set up postgres in production with letsencrypt certificates.
func (p *postgresManager) Create() error {
	db := sqlx.MustConnect("postgres", "postgres://"+p.host+"/postgres?sslmode=disable")
	defer func() {
		err := db.Close()
		if err != nil {
			glog.Fatalf("%+v", errors.Wrapf(err, "failed closing db connection"))
		}
	}()

	if err := p.createUser(db); err != nil {
		return err
	}

	if err := p.createDatabase(db); err != nil {
		return err
	}

	return nil
}

// func (p *postgresManager) Destroy() error {
// 	cmds := []*exec.Cmd{
// 		exec.Command("dropuser", p.username, "--echo"),
// 	}

// 	for _, cmd := range cmds {
// 		runAndLog(cmd)
// 	}

// 	destroyDB(p.username, p.dbname)
// 	return nil
// }

func (p *postgresManager) Migrate() error {
	m, err := migrator.NewMigrator(p.Handle().DB, migrator.Postgres,
		p.assetFunc, p.assetDirFunc)
	if err != nil {
		return err
	}

	if err := m.Migrate(); err != nil {
		return err
	}
	return nil
}

func (p *postgresManager) Handle() *sqlx.DB {
	if db != nil {
		return db
	}

	var err error
	db, err = sqlx.Connect("postgres", p.url)
	if err != nil {
		glog.Fatalf("%+v", errors.Wrapf(err, "failed connecting to postgres DB: %q", p.url))
	}

	return db
}

// func destroyDB(user, dbname string) {
// 	db, err := sql.Open("postgres", fmt.Sprintf("user=%s dbname=%s sslmode=disable", user, dbname))
// 	if err != nil {
// 		log.Fatalf("Error destroying Postgres DB %s: %v", dbname, err)
// 	}
// 	defer db.Close()
// 	db.Exec("update pg_database set datallowconn = 'false' where datname = $1", dbname)
// 	db.Exec("select pg_terminate_backend(pid) from pg_stat_activity where datname = $1", dbname)

// 	cmd := exec.Command("dropdb", dbname, "--echo")
// 	runAndLog(cmd)
// }

var (
	// ErrInternal is returned when an unexpected error occurred.
	ErrInternal = errors.New("internal-error")
)

// Error maps database errors to domain errors
func Error(err error, errorsMap map[string]error) error {
	if err == nil {
		return err
	}

	if err == sql.ErrNoRows {
		return errorsMap["not-found"]
	}

	switch e := err.(type) {
	case *pq.Error:
		if val, ok := errorsMap[e.Constraint]; ok {
			glog.V(2).Infof("%+v", errors.Wrapf(val, "failed db constraint %q: %#v", e.Code.Name(), e))
			return val
		}

		if val, ok := errorsMap[string(e.Code)]; ok {
			glog.V(2).Infof("%+v", errors.Wrapf(val, "failed db constraint %q: %#v", e.Code.Name(), e))
			return val
		}

		if val, ok := errorsMap[e.File]; ok {
			glog.V(2).Infof("%+v", errors.Wrapf(val, "failed db constraint %q: %#v", e.Code.Name(), e))
			return val
		}

		glog.Infof("%#v", e)
		glog.Infof("database error: %+v", errors.Wrap(e, "unexpected postgres error"))
		return ErrInternal
	default:
		glog.Infof("%#v", e)
		glog.Infof("database error: %+v", errors.Wrap(e, "unexpected database error"))
		return ErrInternal
	}
}
