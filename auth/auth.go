package main

import (
	"log"
	"net/http"
	"time"

	"github.com/basilnsage/mwn-ticketapp/auth/errors"
	"github.com/basilnsage/mwn-ticketapp/auth/jwt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	defaultStatus  = "unable to process request"
	defaultCode    = http.StatusBadRequest
	authDB         = "auth"
	authCollection = "users"
)

func main() {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"http://localhost:*"},
		AllowWildcard: true,
		AllowMethods:  []string{"GET", "POST"},
		AllowHeaders:  []string{"Origin", "Content-Type"},
		MaxAge:        12 * time.Hour,
	}))
	// use generic error-handling middleware
	router.Use(errors.HandleErrors())
	UseUserRoutes(router)

	// init mongo cluster connection
	err := InitMongo()
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Print("mongo client initialized")
	defer func() {
		if err := CloseClient(); err != nil {
			panic(err)
		}
	}()

	// init signer
	if err := initSigner(); err != nil {
		log.Fatal(err.Error())
	}
	if err := jwt.InitSigner(); err != nil {
		log.Fatal(err.Error())
	}
	jwt.Sign()

	if err := router.Run(":4000"); err != nil {
		log.Printf("unable to run auth service")
	} else {
		log.Print("gin server running and waiting for requests")
	}
}
