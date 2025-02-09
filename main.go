package main

import (
	"backend/controller"
	"backend/repository"
	"backend/service"
	"database/sql"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	router := gin.Default()

	url, ok := os.LookupEnv("DB_URL")
	if !ok {
		panic(fmt.Errorf("no DB_URL connection link found"))
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DriverName: "postgres",
		Conn:       db,
	}), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	var svc service.PingService = repository.NewPingRepository(gormDB)

	controller.NewPingsController(svc, router.Group("/pings"))

	router.Run("0.0.0.0:3001")
}
