package main

import (
	"context"
	"log"
	"os"
	"time"
)

const (
	dbName = "app"
	collName = "ticket"
)

var (
	InfoLogger *log.Logger
	WarningLogger *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	WarningLogger = log.New(os.Stdout, "WARNING: ", log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	ErrorLogger = log.New(os.Stdout, "ERROR: ", log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
}

func main() {
	// parse environment variables for startup info
	dbConnStr, ok := os.LookupEnv("MONGO_CONN_STR")
	if !ok {
		ErrorLogger.Print("missing mongo connection environment variable: MONGO_CONN_STR")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()
	router, err := newRouter(ctx, dbConnStr, dbName, collName)
	if err != nil {
		ErrorLogger.Printf("could not create new gin router")
		os.Exit(1)
	}

	if err := router.Run(":4000"); err != nil {
		ErrorLogger.Printf("issue running gin router: %v", err)
		os.Exit(1)
	}
}
