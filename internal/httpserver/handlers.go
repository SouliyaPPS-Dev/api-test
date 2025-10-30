package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	authdomain "backoffice/backend/internal/domain/auth"
	productdomain "backoffice/backend/internal/domain/product"
	productusecase "backoffice/backend/internal/usecase/product"
)

func (s *Server) registerRoutes() {
	s.router.Handle("/health", http.HandlerFunc(s.handleHealth))
	s.router.Handle("/auth/register", http.HandlerFunc(s.handleRegister))
	s.router.Handle("/auth/login", http.HandlerFunc(s.handleLogin))
	s.router.Handle("/auth/renew", http.HandlerFunc(s.handleRenewToken))

	authenticated := s.authMiddleware
	s.router.Handle("/products", authenticated(http.HandlerFunc(s.handleProducts)))
	s.router.Handle("/products/", authenticated(http.HandlerFunc(s.handleProductByID)))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	user, err := s.authService.Register(r.Context(), payload.Email, payload.Password, payload.Name)
	if err != nil {
		switch {
		case errors.Is(err, authdomain.ErrEmailExists):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	token, user, err := s.authService.Login(r.Context(), authdomain.Credentials{
		Email:    payload.Email,
		Password: payload.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, authdomain.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid email or password")
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  user,
	})
}

func (s *Server) handleRenewToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	token := extractBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		var payload struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			if errors.Is(err, io.EOF) {
				writeError(w, http.StatusBadRequest, "token required")
			} else {
				writeError(w, http.StatusBadRequest, "invalid JSON payload")
			}
			return
		}
		token = strings.TrimSpace(payload.Token)
	}

	if token == "" {
		writeError(w, http.StatusBadRequest, "token required")
		return
	}

	newToken, err := s.authService.RenewToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, authdomain.ErrTokenInvalid) {
			writeError(w, http.StatusUnauthorized, err.Error())
		} else {
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": newToken,
	})
}

func (s *Server) handleProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		items, err := s.productService.List(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		var payload productusecase.CreateInput
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
		item, err := s.productService.Create(ctx, payload)
		if err != nil {
			switch {
			case errors.Is(err, productdomain.ErrDuplicateSKU):
				writeError(w, http.StatusConflict, err.Error())
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleProductByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/products/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "product id required")
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		item, err := s.productService.Get(ctx, id)
		if err != nil {
			if errors.Is(err, productdomain.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut, http.MethodPatch:
		var payload productusecase.UpdateInput
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
		item, err := s.productService.Update(ctx, id, payload)
		if err != nil {
			switch {
			case errors.Is(err, productdomain.ErrNotFound):
				writeError(w, http.StatusNotFound, err.Error())
			case errors.Is(err, productdomain.ErrDuplicateSKU):
				writeError(w, http.StatusConflict, err.Error())
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := s.productService.Delete(ctx, id); err != nil {
			if errors.Is(err, productdomain.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete)
	}
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r.Header.Get("Authorization"))
		if token == "" {
			writeError(w, http.StatusUnauthorized, "authorization token required")
			return
		}

		user, err := s.authService.VerifyToken(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyUser{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type ctxKeyUser struct{}

func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}
