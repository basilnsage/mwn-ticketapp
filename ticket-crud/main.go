package main

import (
	"github.com/gin-gonic/gin"
	"github.com/nats-io/stan.go"
	"log"
	"os"
	"time"
)

const (
	dbName    = "app"
	collName  = "ticket"
	dbTimeout = 3 * time.Second
)

var (
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

func init() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	WarningLogger = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	ErrorLogger = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
}

func main() {
	// parse environment variables for startup info
	dbConnStr, ok := os.LookupEnv("MONGO_CONN_STR")
	if !ok {
		ErrorLogger.Print("missing mongo connection environment variable: MONGO_CONN_STR")
		os.Exit(1)
	}
	mongoCRUD, err := newCrud(dbTimeout, dbConnStr, dbName, collName)
	if err != nil {
		ErrorLogger.Printf("unable to create DB crud wrapper: %v", err)
		os.Exit(1)
	}
	InfoLogger.Print("able to connect to MongoDB")

	jwtKey, ok := os.LookupEnv("JWT_SIGN_KEY")
	if !ok {
		ErrorLogger.Print("missing JWT HS256 signing key: JWT_SIGN_KEY")
		os.Exit(1)
	}

	natsConnStr, ok := os.LookupEnv("NATS_CONN_STR")
	if !ok {
		ErrorLogger.Printf("missing NATS connection string environment variable: NATS_CONN_STR")
		os.Exit(1)
	}
	natsClient, err := stan.Connect("ticketing", "abc123", stan.NatsURL(natsConnStr))
	if err != nil {
		ErrorLogger.Printf("unable to connect to NATS Streaming Server: %v", err)
		os.Exit(1)
	}
	InfoLogger.Print("able to connect to NATS Streaming Server")
	defer func() {
		if err := natsClient.Close(); err != nil {
			panic(err)
		}
	}()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	//router, err := newRouter(jwtKey, crud)
	server, err := newApiServer(jwtKey, r, mongoCRUD, natsClient)
	if err != nil {
		ErrorLogger.Printf("could not create new API server")
		os.Exit(1)
	}

	if err := server.router.Run(":4000"); err != nil {
		ErrorLogger.Printf("issue running gin router: %v", err)
		os.Exit(1)
	}
}
