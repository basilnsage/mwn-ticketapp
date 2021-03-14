package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/stan.go"
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

func gracefulShutdown(m Closer, n stan.Conn, h *http.Server) (errs []string) {
	// shutdown order: gin -> nats -> mongo
	// allow 30 seconds for each service to shutdown
	httpCtx, httpCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer httpCancel()
	InfoLogger.Print("shutting down the gin HTTP server")
	if err := h.Shutdown(httpCtx); err != nil {
		errs = append(errs, err.Error())
	}

	InfoLogger.Print("shutting down the NATS connection")
	if err := n.Close(); err != nil {
		errs = append(errs, err.Error())
	}

	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer mongoCancel()
	InfoLogger.Print("shutting down the MongoDB connection")
	if err := m.Close(mongoCtx); err != nil {
		errs = append(errs, err.Error())
	}
	return errs
}

type mainConfig map[string]string

func genMainConfig() (mainConfig, []string) {
	var missingEnvs []string
	conf := mainConfig{}
	envToErrString := map[string]string{
		"MONGO_CONN_STR":  "missing mongo connection: MONGO_CONN_STR",
		"JWT_SIGN_KEY":    "missing JWT HS256 signing key: JWT_SIGN_KEY",
		"NATS_CLUSTER_ID": "missing NATS cluster ID: NATS_CLUSTER_ID",
		"NATS_CLIENT_ID":  "missing NATS client ID: NATS_CLIENT_ID",
		"NATS_CONN_STR":   "missing NATS connection string: NATS_CONN_STR",
	}
	for key, errStr := range envToErrString {
		if val, ok := os.LookupEnv(key); ok {
			conf[key] = val
		} else {
			missingEnvs = append(missingEnvs, errStr)
		}
	}
	return conf, missingEnvs
}

func main() {
	// parse environment variables for startup info
	conf, missingEnvs := genMainConfig()
	if len(missingEnvs) > 0 {
		for _, errStr := range missingEnvs {
			ErrorLogger.Print(errStr)
		}
		os.Exit(1)
	}

	// init MongoDB connection
	mongoCRUD, err := newCrud(dbTimeout, conf["MONGO_CONN_STR"], dbName, collName)
	if err != nil {
		ErrorLogger.Printf("unable to create DB crud wrapper: %v", err)
		os.Exit(1)
	}
	InfoLogger.Print("able to connect to MongoDB")

	// init NATS Streaming Server connection
	natsClient, err := stan.Connect(conf["NATS_CLUSTER_ID"], conf["NATS_CLIENT_ID"], stan.NatsURL(conf["NATS_CONN_STR"]))
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

	// create gin router and bind handlers/routes to it
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	server, err := newApiServer(conf["JWT_SIGN_KEY"], r, mongoCRUD, natsClient)
	if err != nil {
		ErrorLogger.Printf("could not create new API server")
		os.Exit(1)
	}
	// start HTTP server and set the gin router as the server handler
	httpServer := &http.Server{
		Addr:    ":4000",
		Handler: server.router,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ErrorLogger.Printf("unable to start HTTP server: %v", err)
			os.Exit(1)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	// SIGINT signal received, begin graceful shutdown
	case <-c:
		cancel()
		InfoLogger.Print("beginning graceful shutdown")
		if errs := gracefulShutdown(mongoCRUD, natsClient, httpServer); len(errs) != 0 {
			ErrorLogger.Printf("unable to perform graceful shutdown:\n%v", strings.Join(errs, "\n"))
			os.Exit(1)
		}
	case <-ctx.Done():
	}
}
