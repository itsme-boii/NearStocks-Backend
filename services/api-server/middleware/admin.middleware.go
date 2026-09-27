package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
)

// RequireAdminKey guards internal endpoints that expose per-user data for arbitrary addresses.
// The caller must send X-Admin-Key matching ADMIN_API_KEY; if ADMIN_API_KEY is unset the route is closed.
func RequireAdminKey(ctx *gin.Context) {
	expected := os.Getenv("ADMIN_API_KEY")
	provided := ctx.GetHeader("X-Admin-Key")
	if expected == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx.Next()
}
