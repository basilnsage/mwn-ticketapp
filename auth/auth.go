package main

import (
	"fmt"
	"github.com/basilnsage/mwn-ticketapp/auth/errors"
	"log"
	"os"
	"time"

	"github.com/basilnsage/mwn-ticketapp/auth/token"
	"github.com/basilnsage/prometheus-gin-metrics"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	authDB         = "auth"
	authCollection = "users"
)

type config struct {
	collection    userColl
	authValidator *token.JWTValidator
}

func main() {
	// init DB connection + configuration
	if err := InitMongo(); err != nil {
		log.Fatalf("unable to create MongoDB connection: %v", err)
	}
	userCollection := GetCollection(GetDatabase(GetClient(), authDB), authCollection)
	defer func() {
		if err := CloseClient(); err != nil {
			panic(err)
		}
	}()

	// init JWT validator struct
	hmacKey, ok := os.LookupEnv("JWT_SIGN_KEY")
	if !ok {
		log.Fatalf("please set the JWT_SIGN_KEY environment variable")
	}
	jwtValidtor, err := token.NewJWTValidator([]byte(hmacKey), "HS256")
	if err != nil {
		log.Fatalf("unable to init JWT Validator: %v", err)
	}

	// bundle the mongo DB collection and jwt parser together into a config
	conf := config{userColl{userCollection}, jwtValidtor}

	// init gin router and init prometheus metric middleware
	metricReg := prometrics.NewRegistry()
	router := gin.Default()

	// config gin
	fmt.Println("set duration middleware")
	router.Use(metricReg.ReportDuration(nil))
	fmt.Println("set error handling middleware")
	router.Use(errors.HandleErrors())
	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"http://localhost:*"},
		AllowWildcard: true,
		AllowMethods:  []string{"GET", "POST"},
		AllowHeaders:  []string{"Origin", "Content-Type"},
		MaxAge:        12 * time.Hour,
	}))

	// use generic error-handling middleware
	UseUserRoutes(router, conf)

	// expose prometheus metrics
	router.GET("/auth/metrics", metricReg.DefaultHandler)

	if err := router.Run(":4000"); err != nil {
		log.Fatalf("unable to run auth service: %v", err)
	} else {
		log.Print("gin server running and waiting for requests")
	}
}
