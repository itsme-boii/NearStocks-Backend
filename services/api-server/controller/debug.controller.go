package controller

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"

	"github.com/gin-gonic/gin"
)

type DebugController struct {
}

func RegisterDebugController(r *gin.RouterGroup) {
	debugController := DebugController{}
	rg := r.Group("/debug")

	rg.GET("/flip/:id", debugController.Flip)
}

type FlipRequest struct {
	Id string `uri:"id" binding:"required"`
}

func (dc *DebugController) Flip(ctx *gin.Context) {
	var req FlipRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		cutils.ApiAbort(ctx, 400, err.Error())
		return
	}
	if len(req.Id) > 2 && req.Id[:2] == "0x" {
		subaccountId, err := cutils.HexToSubaccountId(req.Id)
		if err != nil {
			cutils.ApiAbort(ctx, 400, err.Error())
			return
		}
		cutils.ApiSuccess(ctx, subaccountId, "")
	} else {
		hex, err := cutils.SubaccountIdToHex(req.Id)
		if err != nil {
			cutils.ApiAbort(ctx, 400, err.Error())
			return
		}
		cutils.ApiSuccess(ctx, hex, "")
	}
}
