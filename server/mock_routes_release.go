//go:build !mock

package server

import "github.com/gin-gonic/gin"

// registerMockRoutes is a no-op for release builds.
func registerMockRoutes(r *gin.Engine) {
	// This function is intentionally left empty.
}
