package repository

import (
	"backend/domain"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/netip"
	"testing"
	"time"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
)

var db *sql.DB
var repo pingRepository

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("could not connect to Docker: %s", err)
	}

	resource, err := pool.Run("postgres", "", []string{"POSTGRES_PASSWORD=postgrespass"})
	if err != nil {
		log.Fatalf("could not start resource: %s", err)
	}

	if err := pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", fmt.Sprintf("postgresql://postgres:postgrespass@localhost:%s/postgres?sslmode=disable", resource.GetPort("5432/tcp")))
		if err != nil {
			return err
		}
		repo = pingRepository{db: db}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}

	defer func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}

	}()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	mm, err := migrate.NewWithDatabaseInstance(
		"file:///home/petruekhin/Projects/vkb/backend/db/migration",
		"postgres", driver)
	if err != nil {
		panic(err)
	}
	err = mm.Up()
	if err != nil {
		panic(err)
	}
	m.Run()
}

func TestSomething(t *testing.T) {
	addr, err := netip.ParseAddr("127.0.0.1")
	if err != nil {
		panic(err)
	}

	err = repo.Put(context.Background(), []domain.Ping{{ContainerIP: addr, Timestamp: time.Now(), Success: true}})
	if err != nil {
		panic(err)
	}

	ret, err := repo.Get(context.Background(), PingGetParams{Limit: 10})
	if err != nil {
		panic(err)
	}

	t.Logf("%#v\n", ret)
}
