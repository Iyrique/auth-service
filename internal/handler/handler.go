package handler

import (
	"auth-service/internal/service"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type Handler struct {
	tokenService *service.TokenService
}

func NewHandler(tokenService *service.TokenService) *Handler {
	return &Handler{tokenService: tokenService}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/auth/token", h.issueTokens)     // GET ?user_id=
	mux.HandleFunc("/auth/refresh", h.refreshTokens) // POST
	mux.HandleFunc("/auth/user", h.getUserID)        // GET (protected)
	mux.HandleFunc("/auth/logout", h.logoutUser)     // POST
}

func (h *Handler) issueTokens(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	userAgent := r.UserAgent()
	ip := realIP(r)

	access, err := h.tokenService.GenerateAccessToken(userID)
	if err != nil {
		http.Error(w, "failed to create access token", http.StatusInternalServerError)
		return
	}
	refresh, err := h.tokenService.GenerateRefreshToken(userID, userAgent, ip)
	if err != nil {
		http.Error(w, "failed to create refresh token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"access_token":  access,
		"refresh_token": refresh,
	})
}

func (h *Handler) refreshTokens(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID       string `json:"user_id"`
		RefreshToken string `json:"refresh_token"`
	}
	body, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(body, &req)

	if req.UserID == "" || req.RefreshToken == "" {
		http.Error(w, "user_id and refresh_token required", http.StatusBadRequest)
		return
	}

	access, refresh, err := h.tokenService.RefreshTokens(
		req.UserID, req.RefreshToken, r.UserAgent(), realIP(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"access_token":  access,
		"refresh_token": refresh,
	})
}

func (h *Handler) getUserID(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	userID, err := h.tokenService.ParseAccessToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"user_id": userID})
}

func (h *Handler) logoutUser(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	userID, err := h.tokenService.ParseAccessToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	if err := h.tokenService.InvalidateTokens(userID); err != nil {
		http.Error(w, "logout failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	parts := strings.Split(auth, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	return ""
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

func writeJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}
