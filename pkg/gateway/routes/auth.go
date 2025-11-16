package routes

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	gatewayauth "github.com/synaptica-ai/platform/pkg/gateway/auth"
	"github.com/synaptica-ai/platform/pkg/gateway/middleware"
	"github.com/synaptica-ai/platform/pkg/identity"
)

type AuthHandler struct {
	service     *identity.Service
	tokenSigner *gatewayauth.JWTManager
}

func NewAuthHandler(service *identity.Service, tokenSigner *gatewayauth.JWTManager) *AuthHandler {
	return &AuthHandler{service: service, tokenSigner: tokenSigner}
}

func (h *AuthHandler) Register(r *mux.Router) {
	r.HandleFunc("/bootstrap", h.handleBootstrap).Methods(http.MethodPost)
	r.HandleFunc("/login", h.handleLogin).Methods(http.MethodPost)

	protected := r.NewRoute().Subrouter()
	protected.Use(middleware.Authenticate(h.tokenSigner))
	protected.HandleFunc("/me", h.handleMe).Methods(http.MethodGet)
	protected.HandleFunc("/register", h.handleRegister).Methods(http.MethodPost)
}

func (h *AuthHandler) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	var req models.BootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	org, user, err := h.service.Bootstrap(r.Context(), req)
	if err != nil {
		logger.Log.WithError(err).Warn("bootstrap failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := h.tokenSigner.IssueToken(user)
	if err != nil {
		logger.Log.WithError(err).Error("issue token failed during bootstrap")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, models.AuthResponse{
		Token: token,
		User:  user,
		Org:   org,
	})
}

func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	user, err := h.service.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		logger.Log.WithError(err).Warn("authentication failed")
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.tokenSigner.IssueToken(user)
	if err != nil {
		logger.Log.WithError(err).Error("failed issuing token")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, models.AuthResponse{
		Token: token,
		User:  user,
	})
}

func (h *AuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	claims := r.Context().Value(middleware.UserContextKey)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	authClaims, ok := claims.(*gatewayauth.Claims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	actorUser, err := h.service.GetUser(r.Context(), authClaims.UserID)
	if err != nil {
		logger.Log.WithError(err).Error("failed to load actor user")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	user, err := h.service.RegisterUser(r.Context(), actorUser, req)
	if err != nil {
		logger.Log.WithError(err).Warn("failed to register user")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	authClaims, ok := claims.(*gatewayauth.Claims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.service.GetUser(r.Context(), authClaims.UserID)
	if err != nil {
		logger.Log.WithError(err).Warn("failed to fetch user in /me")
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
