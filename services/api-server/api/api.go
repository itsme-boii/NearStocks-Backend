package api

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/services/api-server/controller"
	"os"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

type ApiServer struct {
	addr string
}

func NewApiServer(addr string) *ApiServer {
	return &ApiServer{addr: addr}
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // FIXME: Handle this for prod
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
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		//ToDo - Add only allowed headers
		AllowedHeaders:   []string{"*"}, // Allow all headers
		AllowCredentials: false,         // clients authenticate with headers, never cookies
	})

	// Apply the CORS middleware to the router
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

	wrg := r.Group("/ws/v1")
	// Register a websocket endpoint
	controller.RegisterOrderbookWebsocket(wrg, &wsUpgrader)
	controller.RegisterUserWebsocket(wrg, &wsUpgrader)
	controller.RegisterQuoteWebsocket(wrg, &wsUpgrader)
	marketPricesWs := controller.RegisterMarketPricesWebsocket(wrg, &wsUpgrader)

	rg := r.Group("api/v1")
	//TODO: testing aws pipelines remove this change later
	controller.RegisterAuthController(rg)
	controller.RegisterNearController(rg)
	controller.RegisterTokenController(rg)
	controller.RegisterSubaccountController(rg)
	controller.RegisterAddressController(rg)
	controller.RegisterOrderController(rg)
	controller.RegisterMarketController(rg)
	controller.RegisterUserController(rg)
	controller.RegisterIPController(rg)
	controller.RegisterRewardsController(rg)
	controller.RegisterPointsController(rg)
	controller.RegisterStatsController(rg)
	controller.RegisterReferralController(rg)
	controller.RegisterStakingController(rg)
	controller.RegisterBatchingController(rg)
	controller.RegisterInsuranceController(rg)
	controller.RegisterDebugController(rg)
	controller.RegisterOptionsController(rg)
	controller.RegisterPreMarketsController(rg, marketPricesWs)
	controller.RegisterSyntheticSpotsController(rg)
	controller.RegisterGovernanceController(rg)
	controller.RegisterAirdropController(rg)
	controller.RegisterKolController(rg)
	controller.RegisterCantonAddressController(rg)
	// Remove this logic when in production
	controller.RegisterBrokerController(rg)
	return r.Run(s.addr)
}
