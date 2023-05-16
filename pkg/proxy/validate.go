/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package proxy

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/v51/github"
)

func validateWebhookSignature(req *http.Request, secret string) error {
	// Parse reads and *verifies* the hook in an inbound request
	if _, err := github.ValidatePayload(req, []byte(secret)); err != nil {
		return errors.New(fmt.Sprintf("validateWebhookSignature: %v", err))
	}
	return nil
}
