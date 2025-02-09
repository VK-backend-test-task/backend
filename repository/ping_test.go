package repository

import (
	"backend/domain"
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	mrand "math/rand/v2"
	"net/netip"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *sql.DB
var repo pingRepository
var mrepo *inMemoryPingRespository
var gormDB *gorm.DB
var sampleData []domain.Ping
var sampleAddresses []netip.Addr

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

	repo = NewPingRepository(gormDB).(pingRepository)
	mrepo = &inMemoryPingRespository{}

	sampleAddresses = make([]netip.Addr, 16)
	for i := range sampleAddresses {
		addrRaw := [4]byte{}
		if n, err := rand.Read(addrRaw[:]); n != 4 || err != nil {
			panic(err)
		}

		addr := netip.AddrFrom4(addrRaw)
		sampleAddresses[i] = addr
	}

	sampleData = make([]domain.Ping, 64)
	tt := time.Now().UTC().Truncate(time.Second)
	for i := range sampleData {
		addr := sampleAddresses[mrand.Int()%len(sampleAddresses)]
		sampleData[i] = domain.Ping{ContainerIP: addr, Timestamp: tt, Success: i%2 == 0}
		tt = tt.Add(time.Second)
	}

	fmt.Println(sampleData)
	fmt.Println()

	m.Run()
}

func checkPingsEqual(p1 []domain.Ping, p2 []domain.Ping, cfg *PingGetParams) error {
	if len(p1) != len(p2) {
		return fmt.Errorf("length mismatch: %d != %d (%#v)", len(p1), len(p2), cfg)
	}

	for i, ping1 := range p1 {
		ping2 := p2[i]
		if ping1.String() != ping2.String() {
			return fmt.Errorf("data mismatch at %d: %s != %s (%#v)", i, ping1, ping2, cfg)
		}
	}

	return nil
}

func checkContainersEqual(p1 []domain.ContainerInfo, p2 []domain.ContainerInfo, cfg *PingAggregateParams) error {
	if len(p1) != len(p2) {
		return fmt.Errorf("length mismatch: %d != %d (%#v)", len(p1), len(p2), cfg)
	}

	for i, ping1 := range p1 {
		ping2 := p2[i]
		if ping1.String() != ping2.String() {
			return fmt.Errorf("data mismatch at %d: %s != %s (%#v)", i, ping1, ping2, cfg)
		}
	}

	return nil
}

func must0(e error) {
	if e != nil {
		panic(e)
	}
}

func must[T any](v T, e error) T {
	if e != nil {
		panic(e)
	}
	return v
}

// compare results to the reference implementation
// func TestPut(t *testing.T) {
// 	must0(repo.Put(context.Background(), sampleData))
// 	must0(mrepo.Put(context.Background(), sampleData))
// 	res := must(repo.Get(context.Background(), PingGetParams{OldestFirst: false}))
// 	mres := must(mrepo.Get(context.Background(), PingGetParams{OldestFirst: false}))
// 	must0(checkPingsEqual(res, mres, nil))
// }

/*
// compare all possible options of these two implementations
func TestGet(t *testing.T) {
	must0(repo.Put(context.Background(), sampleData))
	must0(mrepo.Put(context.Background(), sampleData))

	// orders := []domain.ContainerOrder{domain.ContainerSortAsc, domain.ContainerSortDesc}
	// props := []domain.ContainerSortProperty{domain.ContainerSortByIP, domain.ContainerSortByLastPing, domain.ContainerSortByLastSuccess}
	// for _, order := range orders {
	// 	for _, prop := range props {
	bools := []bool{false, true}
	fls := false
	tru := true
	pbools := []*bool{nil, &fls, &tru}
	cips := []*netip.Addr{nil}
	for _, sample := range sampleAddresses {
		cips = append(cips, &sample)
	}

	for limit := 0; limit < 8; limit++ {
		for offset := 0; offset < 8; offset++ {
			for _, success := range pbools {
				for _, oldestFirst := range bools {
					for _, containerIP := range cips {
						params := PingGetParams{
							ContainerIP: containerIP,
							OldestFirst: oldestFirst,
							Success:     success,
							Limit:       limit,
							Offset:      offset,
						}
						res := must(repo.Get(context.Background(), params))
						mres := must(mrepo.Get(context.Background(), params))
						must0(checkPingsEqual(res, mres))
					}
				}
			}
		}
	}
}
*/

// compare all possible options of these two implementations
func TestAggregate(t *testing.T) {
	must0(repo.Put(context.Background(), sampleData))
	must0(mrepo.Put(context.Background(), sampleData))

	orders := []domain.ContainerOrder{domain.ContainerSortAsc, domain.ContainerSortDesc}
	props := []domain.ContainerSortProperty{domain.ContainerSortByLastPing, domain.ContainerSortByLastSuccess}
	tt := sampleData[0].Timestamp.Add(-1 * time.Second)
	ttt := sampleData[len(sampleData)-1].Timestamp.Add(1 * time.Second)
	pbefs := []*time.Time{nil, &tt, &sampleData[0].Timestamp, &sampleData[1].Timestamp, &sampleData[len(sampleData)-2].Timestamp, &sampleData[len(sampleData)-1].Timestamp, &ttt}

	for _, order := range orders {
		for _, prop := range props {
			for _, pbef := range pbefs {
				for limit := 0; limit < 8; limit++ {
					for offset := 0; offset < 8; offset++ {
						params := PingAggregateParams{
							SortProperty: prop,
							SortOrder:    order,
							Limit:        limit,
							Offset:       offset,
							PingBefore:   pbef,
						}
						res := must(repo.Aggregate(context.Background(), params))
						mres := must(mrepo.Aggregate(context.Background(), params))
						err := checkContainersEqual(res, mres, &params)
						if err != nil {
							fmt.Println(res)
							fmt.Println()
							fmt.Println(mres)
							panic(err)
						}
					}
				}
			}
		}
	}
}
