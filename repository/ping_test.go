package repository

import (
	"backend/domain"
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"net/netip"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *sql.DB
var repo PingRepository
var gormDB *gorm.DB

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
		err = db.Ping()
		return err
	}); err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}

	defer func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}

	}()

	gormDB, err = gorm.Open(postgres.New(postgres.Config{
		DriverName: "postgres",
		Conn:       db,
	}), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	repo = NewPingRepository(gormDB)

	m.Run()
}

func TestPut(t *testing.T) {
	records := make([]domain.Ping, 1024)
	for i := range records {
		addrRaw := [4]byte{}
		if n, err := rand.Read(addrRaw[:]); n != 4 || err != nil {
			panic(err)
		}

		addr := netip.AddrFrom4(addrRaw)

		records[i] = domain.Ping{ContainerIP: addr, Timestamp: time.Now().UTC(), Success: true}
	}

	err := repo.Put(context.Background(), records)
	if err != nil {
		panic(err)
	}

	gormPings := make([]gormPingModel, 0)
	gormDB.Model(&gormPingModel{}).Limit(len(records) + 1).Find(&gormPings)
	if len(gormPings) != len(records) {
		t.Fatalf("len mismatch (received %d instead of %d)", len(gormPings), len(records))
	}

	for i, gormPing := range gormPings {
		if gormPing.ContainerIP != records[i].ContainerIP.String() {
			t.Fatalf("IP addr mismatch at %d (our %s, their %s)", i, records[i].ContainerIP.String(), gormPing.ContainerIP)
		}

		if gormPing.Timestamp != records[i].Timestamp.Format(time.RFC3339) {
			t.Fatalf("time mismatch at %d (our %s, their %s)", i, records[i].Timestamp.Format(time.RFC3339), gormPing.Timestamp)
		}

		if gormPing.Success != records[i].Success {
			t.Fatalf("success mismatch at %d (out %t, their %t)", i, records[i].Success, gormPing.Success)
		}
	}
}
