//go:build mock

package server

import "github.com/gin-gonic/gin"

// registerMockRoutes adds the mock data endpoint to the router.
func registerMockRoutes(r *gin.Engine) {
	r.GET("/api/mock-data", mockDataHandler)
}
