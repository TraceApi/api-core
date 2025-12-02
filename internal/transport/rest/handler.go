/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

package rest

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/TraceApi/api-core/internal/core/domain"
	"github.com/TraceApi/api-core/internal/core/ports"
	"github.com/TraceApi/api-core/internal/transport/rest/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
	r.Get("/passports", h.ListPassports)
	r.Put("/passports/{id}", h.UpdatePassport)
	r.Post("/passports/{id}/publish", h.PublishPassport)
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
	manufacturerName, ok := middleware.GetManufacturerName(r.Context())
	if !ok || manufacturerName == "" {
		// Fallback to ID if name is missing (should be handled by middleware, but safe guard)
		manufacturerName = manufacturerID
	}
	passport, err := h.service.CreatePassport(r.Context(), manufacturerID, manufacturerName, category, body)
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
	json.NewEncoder(w).Encode(passport)
}

// PublishPassport handles POST /passports/{id}/publish
func (h *PassportHandler) PublishPassport(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid passport id", http.StatusBadRequest)
		return
	}

	passport, err := h.service.PublishPassport(r.Context(), id)
	if err != nil {
		h.log.Error("failed to publish passport", "error", err)
		if errors.Is(err, domain.ErrPassportAlreadyPublished) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) || strings.Contains(err.Error(), "validation failed") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(passport)
}

// ListPassports handles GET /passports
func (h *PassportHandler) ListPassports(w http.ResponseWriter, r *http.Request) {
	// 1. Get Manufacturer ID from Context
	manufacturerID, ok := middleware.GetManufacturerID(r.Context())
	if !ok {
		http.Error(w, "unauthorized: missing manufacturer identity", http.StatusUnauthorized)
		return
	}

	// 2. Call Service
	passports, err := h.service.ListPassports(r.Context(), manufacturerID)
	if err != nil {
		h.log.Error("failed to list passports", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// 3. Respond
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(passports)
}

// UpdatePassport handles PUT /passports/{id}
func (h *PassportHandler) UpdatePassport(w http.ResponseWriter, r *http.Request) {
	// 1. Get Manufacturer ID
	manufacturerID, ok := middleware.GetManufacturerID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Parse ID
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid passport id", http.StatusBadRequest)
		return
	}

	// 3. Read Body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 4. Call Service
	passport, err := h.service.UpdatePassport(r.Context(), id, manufacturerID, body)
	if err != nil {
		h.log.Error("failed to update passport", "error", err)
		if errors.Is(err, domain.ErrInvalidInput) || strings.Contains(err.Error(), "validation failed") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Respond
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(passport)
}
