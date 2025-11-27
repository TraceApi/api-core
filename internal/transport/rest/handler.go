/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-21
 * Change License: AGPL-3.0
 */

package rest

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/TraceApi/api-core/internal/core/domain"
	"github.com/TraceApi/api-core/internal/core/ports"
	"github.com/TraceApi/api-core/internal/transport/rest/middleware"
	"github.com/go-chi/chi/v5"
)

type PassportHandler struct {
	service ports.PassportService
	log     *slog.Logger
}

func NewPassportHandler(s ports.PassportService, log *slog.Logger) *PassportHandler {
	return &PassportHandler{service: s, log: log}
}

// RegisterRoutes wires up the endpoints to the router
func (h *PassportHandler) RegisterRoutes(r chi.Router) {
	r.Post("/passports", h.CreatePassport)
}

// CreatePassport handles POST /passports?category=BATTERY_INDUSTRIAL
func (h *PassportHandler) CreatePassport(w http.ResponseWriter, r *http.Request) {
	// 1. Parse Query Param for Category
	catParam := r.URL.Query().Get("category")
	if catParam == "" {
		http.Error(w, "missing 'category' query parameter", http.StatusBadRequest)
		return
	}
	category := domain.ProductCategory(catParam)

	// 2. Get Manufacturer ID from Context (set by AuthMiddleware)
	manufacturerID, ok := middleware.GetManufacturerID(r.Context())
	if !ok {
		// Should be caught by middleware, but safe guard here
		http.Error(w, "unauthorized: missing manufacturer identity", http.StatusUnauthorized)
		return
	}

	// 3. Read Body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 4. Call Service
	passport, err := h.service.CreatePassport(r.Context(), manufacturerID, category, body)
	if err != nil {
		h.log.Error("failed to create passport", "error", err)

		if errors.Is(err, domain.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// 5. Respond
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(passport); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
