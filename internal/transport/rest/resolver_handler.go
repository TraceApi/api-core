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
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/TraceApi/api-core/internal/core/ports"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
)

type ResolverHandler struct {
	service ports.PassportService // We reuse the service for reading
	log     *slog.Logger
}

func NewResolverHandler(s ports.PassportService, log *slog.Logger) *ResolverHandler {
	return &ResolverHandler{service: s, log: log}
}

func (h *ResolverHandler) RegisterResolverRoutes(r chi.Router) {
	// The Short URL route (e.g., tapi.eu/r/123)
	r.Get("/r/{id}", h.ResolvePassport)
	r.Get("/r/{id}/qr", h.GetQRCode)
}

func (h *ResolverHandler) ResolvePassport(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	uid, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid Passport ID", http.StatusBadRequest)
		return
	}

	// 1. Fetch Data (Using the Read method we added to the service implicitly via Repo)
	// We need to expose a GetPassport method in the Service interface first!
	// (We will do that in the next step)
	passport, err := h.service.GetPassport(r.Context(), uid)
	if err != nil {
		h.log.Warn("passport not found", "id", uid, "error", err)
		http.Error(w, "Passport Not Found", http.StatusNotFound)
		return
	}

	// 2. Content Negotiation (The "Smart" Part)
	acceptHeader := r.Header.Get("Accept")

	if strings.Contains(acceptHeader, "text/html") {
		// --- RETURN HTML (Browser) ---
		w.Header().Set("Content-Type", "text/html")

		// In a real app, use html/template here.
		// For MVP, we inject the data into a simple string.
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>TraceApi Passport</title>
				<meta name="viewport" content="width=device-width, initial-scale=1">
				<style>
					body { font-family: sans-serif; padding: 20px; max-width: 600px; margin: 0 auto; }
					.card { border: 1px solid #ddd; border-radius: 8px; padding: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
					.status { display: inline-block; padding: 4px 8px; border-radius: 4px; background: #e0f7fa; color: #006064; font-size: 0.8em; font-weight: bold;}
					h1 { font-size: 1.2em; margin-top: 0; }
					pre { background: #f5f5f5; padding: 10px; overflow-x: auto; border-radius: 4px;}
				</style>
			</head>
			<body>
				<div class="card">
					<span class="status">%s</span>
					<h1>Product Passport</h1>
					<p><strong>ID:</strong> %s</p>
					<p><strong>Category:</strong> %s</p>
					<hr/>
					<h3>Technical Data</h3>
					<pre>%s</pre>
				</div>
			</body>
			</html>
		`, passport.Status, passport.ID, passport.ProductCategory, passport.Attributes)

		w.Write([]byte(html))

	} else {
		// --- RETURN JSON (API/App) ---
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(passport)
	}
}

func (h *ResolverHandler) GetQRCode(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	// 1. Construct the Public URL
	// In Prod, this comes from ENV VAR (e.g., "https://tapi.eu")
	// In Dev, we use localhost or your LAN IP
	baseURL := "http://localhost:8081"
	targetURL := fmt.Sprintf("%s/r/%s", baseURL, idStr)

	// 2. Generate QR Code (Recovery Level M is standard for industrial)
	// 256 is the size in pixels (ignored for SVG, but required by API)
	var png []byte
	var err error

	// Check if they want PNG or SVG? Let's default to PNG for easy testing,
	// but if ?format=svg is passed, return SVG.
	format := r.URL.Query().Get("format")

	if format == "svg" {
		// Logic for SVG (go-qrcode supports it differently, but for MVP let's stick to PNG first
		// to keep code simple, or use the library's Write method)
		// Actually, for simplicity, let's serve PNG to start.
		png, err = qrcode.Encode(targetURL, qrcode.Medium, 256)
	} else {
		png, err = qrcode.Encode(targetURL, qrcode.Medium, 256)
	}

	if err != nil {
		h.log.Error("failed to generate qr", "error", err)
		http.Error(w, "Failed to generate QR", http.StatusInternalServerError)
		return
	}

	// 3. Return Image
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}
