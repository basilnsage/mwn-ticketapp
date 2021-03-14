package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/stan.go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	dbName               = "app"
	ticketCollectionName = "tickets"
	ordersCollectionName = "orders"
	dbTimeout            = 3 * time.Second
)

// used env vars
const (
	mongoConnStr  = "MONGO_CONN_STR"
	jwtSignKey    = "JWT_SIGN_KEY"
	natsClusterId = "NATS_CLUSTER_ID"
	natsClientId  = "NATS_CLIENT_ID"
	natsConnStr   = "NATS_CONN_STR"
)

var (
	InfoLogger *log.Logger
	//WarningLogger *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	//WarningLogger = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	ErrorLogger = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
}

type groupCloser struct {
	httpServer  *http.Server
	stan        stan.Conn
	mongoClient *mongo.Client
	listener    natsListener
}

func (gc groupCloser) shutdown(code int) {
	// shutdown order: gin -> nats -> mongo
	// allow 90 seconds for everything to shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	InfoLogger.Print("shutting down the gin HTTP server")
	if gc.httpServer != nil {
		if err := gc.httpServer.Shutdown(ctx); err != nil {
			panic(err)
		}
	}

	InfoLogger.Print("shutting down the NATS listener")
	if err := gc.listener.close(); err != nil {
		panic(err)
	}

	InfoLogger.Print("shutting down the NATS connection")
	if gc.stan != nil {
		if err := gc.stan.Close(); err != nil {
			panic(err)
		}
	}

	InfoLogger.Print("shutting down the MongoDB connection")
	if gc.mongoClient != nil {
		if err := gc.mongoClient.Disconnect(ctx); err != nil {
			panic(err)
		}
	}

	InfoLogger.Print("all service connections shut down")
	os.Exit(code)
}

func validateEnvVars() []string {
	var missingEnvs []string
	envToErrString := map[string]string{
		mongoConnStr:  "missing mongo connection: MONGO_CONN_STR",
		jwtSignKey:    "missing JWT HS256 signing key: JWT_SIGN_KEY",
		natsClusterId: "missing NATS cluster ID: NATS_CLUSTER_ID",
		natsClientId:  "missing NATS client ID: NATS_CLIENT_ID",
		natsConnStr:   "missing NATS connection string: NATS_CONN_STR",
	}
	for key, errStr := range envToErrString {
		if _, ok := os.LookupEnv(key); !ok {
			missingEnvs = append(missingEnvs, errStr)
		}
	}
	return missingEnvs
}

func main() {
	// for handling graceful shutdown of all required services
	gc := groupCloser{}

	// parse environment variables for startup info
	missingEnvs := validateEnvVars()
	if len(missingEnvs) > 0 {
		for _, errStr := range missingEnvs {
			ErrorLogger.Print(errStr)
		}
		os.Exit(1)
	}

	// init MongoDB collections
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	client, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv(mongoConnStr)))
	if err != nil {
		ErrorLogger.Printf("mongo.NewClient: %v", err)
		gc.shutdown(1)
	}
	if err := client.Connect(ctx); err != nil {
		ErrorLogger.Printf("mongo client.Connect: %v", err)
		gc.shutdown(1)
	}
	if err := client.Ping(ctx, nil); err != nil {
		ErrorLogger.Printf("mongo client.Ping: %v", err)
		gc.shutdown(1)
	}
	InfoLogger.Print("connected to MongoDB")
	gc.mongoClient = client
	db := client.Database(dbName)

	ticketCollection := newTicketCollection(db.Collection(ticketCollectionName), dbTimeout)
	ordersCollection := newOrdersCollection(db.Collection(ordersCollectionName), dbTimeout)

	// init NATS Streaming Server connection
	natsConn, err := stan.Connect(os.Getenv(natsClusterId), os.Getenv(natsClientId), stan.NatsURL(os.Getenv(natsConnStr)))
	if err != nil {
		ErrorLogger.Printf("natsConn.Connect: %v", err)
		gc.shutdown(1)
	}
	InfoLogger.Print("connected to NATS Streaming Server")
	gc.stan = natsConn

	// create gin router and bind handlers/routes to it
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	server, err := newApiServer(os.Getenv(jwtSignKey), 15*time.Minute, r, ticketCollection, ordersCollection, natsConn)
	if err != nil {
		ErrorLogger.Printf("could not create new API server")
		gc.shutdown(1)
		return // this will never be called but it makes the IDE happy
	}

	natsEventListener := newNatsListener(ticketCollection, natsConn)
	gc.listener = natsEventListener
	if err := natsEventListener.listenForEvents(); err != nil {
		ErrorLogger.Printf("failed to start event listener: %v", err)
		gc.shutdown(1)
	}

	// start HTTP server and set the gin router as the server handler
	httpServer := &http.Server{
		Addr:    ":4000",
		Handler: server.router,
	}
	gc.httpServer = httpServer
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ErrorLogger.Printf("unable to start HTTP server: %v", err)
			gc.shutdown(1)
		}
	}()

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	// SIGINT signal received, begin graceful shutdown
	case <-c:
		InfoLogger.Print("beginning graceful shutdown")
		gc.shutdown(0)
	case <-ctx.Done():
	}
}
