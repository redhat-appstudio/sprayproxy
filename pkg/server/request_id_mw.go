/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package server

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Middleware adding request ID to gin context.
// Note that this is a simple unique ID that can be used for debugging purposes.
// In the future, this might be replaced with OpenTelemetry IDs/tooling.
func addRequestId() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("requestId", uuid.New().String())
		c.Next()
	}
}
