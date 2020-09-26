package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/basilnsage/mwn-ticketapp/auth/errors"
	"github.com/basilnsage/mwn-ticketapp/auth/token"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	defaultStatus  = "unable to process request"
	defaultCode    = http.StatusBadRequest
	authDB         = "auth"
	authCollection = "users"
)

type config struct {
	collection userColl
	authValidator *token.JWTValidator
}

func main() {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"http://localhost:*"},
		AllowWildcard: true,
		AllowMethods:  []string{"GET", "POST"},
		AllowHeaders:  []string{"Origin", "Content-Type"},
		MaxAge:        12 * time.Hour,
	}))

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

	// bundle the mongo DB collection and jwt parser together
	conf := config{userColl{userCollection}, jwtValidtor}

	// use generic error-handling middleware
	router.Use(errors.HandleErrors())
	UseUserRoutes(router, conf)

	// init mongo cluster connection

	if err := router.Run(":4000"); err != nil {
		log.Fatalf("unable to run auth service: %v", err)
	} else {
		log.Print("gin server running and waiting for requests")
	}
}
