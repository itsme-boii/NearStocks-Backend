package api

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/services/engine/controller"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
)

type ApiServer struct {
	addr string
}

func NewApiServer(addr string) *ApiServer {
	return &ApiServer{addr: addr}
}

// 1. Registers all the services
// 2. Listens and serve on address
func (s *ApiServer) Run() error {
	if os.Getenv("IS_DEV") != "1" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// CORS middleware configuration to allow all origins and headers
	corsConfig := cors.New(cors.Options{
		AllowedOrigins: cutils.CORSAllowedOrigins(),
		AllowedMethods: []string{"POST", "OPTIONS"},
		//ToDo - Add only allowed headers
		AllowedHeaders:   []string{"*"}, // Allow all headers
		AllowCredentials: false,         // clients authenticate with headers, never cookies
	})

	// Apply the CORS middleware to the router using gin's middleware functionality
	r.Use(func(c *gin.Context) {
		corsConfig.HandlerFunc(c.Writer, c.Request)
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	rg := r.Group("/api/v1")
	controller.RegisterMessageController(rg)

	return r.Run(s.addr)
}
