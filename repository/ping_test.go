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

	"github.com/ory/dockertest/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *sql.DB
var repo PingRepository

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
		url := fmt.Sprintf("postgresql://postgres:postgrespass@localhost:%s/postgres?sslmode=disable", resource.GetPort("5432/tcp"))
		db, err = sql.Open("postgres", url)
		if err != nil {
			return err
		}

		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}

	defer func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}

	}()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DriverName: "postgres",
		Conn:       db,
	}), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	repo = NewPingRepository(gormDB)

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
