package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

func main() {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:*"},
		AllowWildcard: true,
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Origin", "Content-Type"},
		MaxAge: 12 * time.Hour,
	}))
	router.GET("/api/users/currentUser", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello there")
	})
	if err := router.Run(":4000"); err != nil {
		log.Printf("unable to run auth service")
	}
}