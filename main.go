package main

import (
	"backend/controller"
	"backend/repository"
	"backend/service"
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	router := gin.Default()

	url := fmt.Sprintf("postgresql://postgres:postgrespass@localhost:%s/postgres?sslmode=disable", resource.GetPort("5432/tcp"))
	db, err := sql.Open("postgres", url)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	if err != nil {
		panic(fmt.Errorf("Could not connect to database: %s", err))
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DriverName: "postgres",
		Conn:       db,
	}), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	var svc service.PingService = repository.NewPingRepository(gormDB)

	controller.NewPingsController(svc, router.Group("/pings")
)
}
