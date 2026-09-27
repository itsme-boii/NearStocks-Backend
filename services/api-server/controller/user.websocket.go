package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type UserWebsocket struct {
	wsUpgrader *websocket.Upgrader
}

func (u *UserWebsocket) upgradeWS(ctx *gin.Context) (*websocket.Conn, error) {
	return u.wsUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
}

func RegisterUserWebsocket(rg *gin.RouterGroup, wsUpgrader *websocket.Upgrader) {
	userWebsocket := UserWebsocket{
		wsUpgrader: wsUpgrader,
	}

	rg.GET("/userstate", userWebsocket.GetStateData)
}

func (u *UserWebsocket) GetStateData(ctx *gin.Context) {
	conn, err := u.upgradeWS(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	defer conn.Close()

	newCtx, cancel := context.WithCancel(ctx.Request.Context())
	defer cancel()

	go func() {
		defer cancel()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				xlog.Errorf("User Websocket - Client disconnected")
				return
			}
		}
	}()

	for {
		select {
		case <-newCtx.Done():
			xlog.Infof("User Websocket - Stopping state data update due to client disconnect")
			return
		default:

			// Fetch latest trade history from database
			tradeHistory, err := (&db.FillDB{}).GetLatestTradeHistory()
			if err != nil {
				conn.WriteJSON(gin.H{"error": err.Error()})
				return
			}

			stateData := make(map[string]gin.H)
			if tradeHistory != nil && len(*tradeHistory) > 0 {
				for _, trade := range *tradeHistory {
					ProductID := trade.MarketId
					lastExecutedTrade := gin.H{
						"Amountx18": trade.Amountx18.Val.String(),
						"Pricex18":  trade.Amountx18.Val.String(),
						"Timestamp": trade.CreatedAt,
						"type":      trade.Side,
					}
					stateData[fmt.Sprintf("%d", ProductID)] = gin.H{
						"lastExecutedTrade": lastExecutedTrade,
					}
				}
			}

			response := gin.H{
				"stateData": stateData,
			}

			err = conn.WriteJSON(response)
			if err != nil {
				xlog.Errorf("User Websocket - Error sending state data to client:", err)
				return
			}
			time.Sleep(1 * time.Second)
		}
	}
}
