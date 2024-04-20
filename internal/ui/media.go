// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"
	"time"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/http/response/html"
)

func (h *handler) showMedium(w http.ResponseWriter, r *http.Request) {
	mediumID := request.RouteInt64Param(r, "mediumID")
	medium, err := h.store.MediumByID(mediumID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if medium == nil {
		html.NotFound(w, r)
		return
	}

	response.New(w, r).WithCaching(medium.Hash, 72*time.Hour, func(b *response.Builder) {
		b.WithHeader("Content-Security-Policy", `default-src 'self'`)
		b.WithHeader("Content-Type", medium.MimeType)
		b.WithBody(medium.Content)
		b.WithoutCompression()
		b.Write()
	})
}

func (h *handler) showMediumText(w http.ResponseWriter, r *http.Request) {
	mediumID := request.RouteInt64Param(r, "mediumID")
	medium, err := h.store.MediumByID(mediumID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if medium == nil {
		html.NotFound(w, r)
		return
	}

	response.New(w, r).WithCaching(medium.Hash, 72*time.Hour, func(b *response.Builder) {
		b.WithHeader("Content-Security-Policy", `default-src 'self'`)
		b.WithHeader("Content-Type", "text/plain; charset=UTF-8")
		b.WithBody(medium.Text)
		b.WithoutCompression()
		b.Write()
	})
}
