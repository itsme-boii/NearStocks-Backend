package controller

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"net/http"

	"github.com/gin-gonic/gin"
)

type BrokerController struct {
}

func RegisterBrokerController(
	r *gin.RouterGroup,
) {
	brokerController := BrokerController{}
	rg := r.Group("/broker")

	// Endpoints
	rg.POST("", brokerController.Create)
	rg.PUT("/:id", brokerController.Update)
	rg.GET("/:id", brokerController.GetById)
	rg.GET("", brokerController.GetAll)
}

type BrokerCreateRequest struct {
	Name string `json:"name" binding:"required"`
}

func (*BrokerController) Create(ctx *gin.Context) {
	var requestObj BrokerCreateRequest
	if err := ctx.BindJSON(&requestObj); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	broker := (&db.BrokerDB{}).Create(requestObj.Name)
	cutils.ApiResponse(ctx, gin.H{"broker": broker}, http.StatusOK, "")
}

type BrokerUpdateRequestURI struct {
	BrokerId uint `uri:"id" binding:"required"`
}
type BrokerUpdateRequestBody struct {
	Name string `json:"name" binding:"required"`
}

type BodyUpdateRequestObj struct {
	BrokerUpdateRequestURI
	BrokerUpdateRequestBody
}

func (*BrokerController) Update(ctx *gin.Context) {
	var _requestUri BrokerUpdateRequestURI
	var _requestBody BrokerUpdateRequestBody

	if err := ctx.BindUri(&_requestUri); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if err := ctx.BindJSON(&_requestBody); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	requestObj := BodyUpdateRequestObj{BrokerUpdateRequestBody: _requestBody, BrokerUpdateRequestURI: _requestUri}
	broker := (&db.BrokerDB{}).Update(requestObj.BrokerId, requestObj.Name)
	if broker == nil {
		ctx.AbortWithError(http.StatusNotFound, fmt.Errorf("broker with id '%v' not found", requestObj.BrokerId))
		return
	}
	cutils.ApiResponse(ctx, gin.H{"broker": broker}, http.StatusOK, "")
}

type BrokerGetByIdRequest struct {
	Id uint `uri:"id" binding:"required"`
}

func (*BrokerController) GetById(ctx *gin.Context) {
	var requestObj BrokerGetByIdRequest
	if err := ctx.BindUri(&requestObj); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	broker := (&db.BrokerDB{}).GetById(requestObj.Id)
	if broker == nil {
		ctx.AbortWithError(http.StatusNotFound, fmt.Errorf("broker with id:%v not found", requestObj.Id))
		return
	}

	cutils.ApiResponse(ctx, gin.H{"broker": broker}, http.StatusOK, "")
}

func (*BrokerController) GetAll(ctx *gin.Context) {
	brokers := (&db.BrokerDB{}).GetAll()
	cutils.ApiResponse(ctx, gin.H{"brokers": brokers}, http.StatusOK, "")
}
