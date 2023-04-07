/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package server

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Middleware adding request ID to gin context
func addRequestId() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("requestId", uuid.New().String())
		c.Next()
	}
}
