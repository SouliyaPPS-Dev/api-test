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
	userusecase "backoffice/backend/internal/usecase/user"
)

func (s *Server) registerRoutes() {
	s.router.Handle("/health", http.HandlerFunc(s.handleHealth))
	s.router.Handle("/auth/register", http.HandlerFunc(s.handleRegister))
	s.router.Handle("/auth/login", http.HandlerFunc(s.handleLogin))
	s.router.Handle("/auth/renew", http.HandlerFunc(s.handleRenewToken))

	authenticated := s.authMiddleware
	s.router.Handle("/products", authenticated(http.HandlerFunc(s.handleProducts)))
	s.router.Handle("/products/", authenticated(http.HandlerFunc(s.handleProductByID)))
	s.router.Handle("/users/change-password", authenticated(http.HandlerFunc(s.handleChangePassword)))
	s.router.Handle("/users/me/role", authenticated(http.HandlerFunc(s.handleUserRole)))
	s.router.Handle("/admin/users", authenticated(http.HandlerFunc(s.handleAdminUsers)))
	s.router.Handle("/admin/users/", authenticated(http.HandlerFunc(s.handleAdminUserByID)))
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

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	user, ok := currentUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var payload struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "current_password and new_password required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid JSON payload")
		}
		return
	}

	if err := s.authService.ChangePassword(r.Context(), user.ID, payload.CurrentPassword, payload.NewPassword); err != nil {
		switch {
		case errors.Is(err, authdomain.ErrPasswordMismatch):
			writeError(w, http.StatusBadRequest, "current password is incorrect")
		case errors.Is(err, authdomain.ErrPasswordUnchanged):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, authdomain.ErrUserNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUserRole(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"user": user,
		})
	case http.MethodPut, http.MethodPatch:
		var payload struct {
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			if errors.Is(err, io.EOF) {
				writeError(w, http.StatusBadRequest, "role is required")
			} else {
				writeError(w, http.StatusBadRequest, "invalid JSON payload")
			}
			return
		}

		role := strings.TrimSpace(payload.Role)
		if role == "" {
			writeError(w, http.StatusBadRequest, "role is required")
			return
		}

		normalized := strings.ToLower(role)
		if normalized == string(authdomain.RoleAdmin) && user.Role != authdomain.RoleAdmin {
			writeError(w, http.StatusForbidden, "insufficient privileges to assign admin role")
			return
		}

		user, err := s.userService.Update(r.Context(), user.ID, userusecase.UpdateInput{
			Role: &role,
		})
		if err != nil {
			switch {
			case errors.Is(err, authdomain.ErrInvalidRole):
				writeError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, authdomain.ErrUserNotFound):
				writeError(w, http.StatusNotFound, err.Error())
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"user": user,
		})
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPut, http.MethodPatch)
	}
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !s.requireAdmin(w, r) {
			return
		}
		filter := userusecase.Filter{
			Role: r.URL.Query().Get("role"),
		}
		users, err := s.userService.List(r.Context(), filter)
		if err != nil {
			if errors.Is(err, authdomain.ErrInvalidRole) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": users})
	case http.MethodPost:
		if !s.requireAdmin(w, r) {
			return
		}
		var payload struct {
			Email    string `json:"email"`
			Name     string `json:"name"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			if errors.Is(err, io.EOF) {
				writeError(w, http.StatusBadRequest, "email, password, and role are required")
			} else {
				writeError(w, http.StatusBadRequest, "invalid JSON payload")
			}
			return
		}

		user, err := s.userService.Create(r.Context(), userusecase.CreateInput{
			Email:    payload.Email,
			Name:     payload.Name,
			Password: payload.Password,
			Role:     payload.Role,
		})
		if err != nil {
			switch {
			case errors.Is(err, authdomain.ErrEmailExists):
				writeError(w, http.StatusConflict, err.Error())
			case errors.Is(err, authdomain.ErrInvalidRole):
				writeError(w, http.StatusBadRequest, err.Error())
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"user": user})
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleAdminUserByID(w http.ResponseWriter, r *http.Request) {
	remainder := strings.TrimPrefix(r.URL.Path, "/admin/users/")
	remainder = strings.TrimSpace(remainder)
	remainder = strings.Trim(remainder, "/")
	if remainder == "" {
		writeError(w, http.StatusBadRequest, "user id required")
		return
	}

	segments := strings.Split(remainder, "/")
	id := strings.TrimSpace(segments[0])
	if id == "" {
		writeError(w, http.StatusBadRequest, "user id required")
		return
	}

	if len(segments) > 1 {
		switch strings.TrimSpace(segments[1]) {
		case "role":
			s.handleAdminUserRole(w, r, id)
		default:
			writeError(w, http.StatusNotFound, "resource not found")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		if !s.requireAdmin(w, r) {
			return
		}
		user, err := s.userService.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, authdomain.ErrUserNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
			} else {
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		writeJSON(w, http.StatusOK, user)
	case http.MethodPut, http.MethodPatch:
		if !s.requireAdmin(w, r) {
			return
		}
		var payload struct {
			Email *string `json:"email"`
			Name  *string `json:"name"`
			Role  *string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			if errors.Is(err, io.EOF) {
				writeError(w, http.StatusBadRequest, "update payload required")
			} else {
				writeError(w, http.StatusBadRequest, "invalid JSON payload")
			}
			return
		}

		user, err := s.userService.Update(r.Context(), id, userusecase.UpdateInput{
			Email: payload.Email,
			Name:  payload.Name,
			Role:  payload.Role,
		})
		if err != nil {
			switch {
			case errors.Is(err, authdomain.ErrUserNotFound):
				writeError(w, http.StatusNotFound, err.Error())
			case errors.Is(err, authdomain.ErrEmailExists):
				writeError(w, http.StatusConflict, err.Error())
			case errors.Is(err, authdomain.ErrInvalidRole):
				writeError(w, http.StatusBadRequest, err.Error())
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		writeJSON(w, http.StatusOK, user)
	case http.MethodDelete:
		if !s.requireAdmin(w, r) {
			return
		}
		if err := s.userService.Delete(r.Context(), id); err != nil {
			if errors.Is(err, authdomain.ErrUserNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
			} else {
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete)
	}
}

func (s *Server) handleAdminUserRole(w http.ResponseWriter, r *http.Request, userID string) {
	switch r.Method {
	case http.MethodGet:
		if !s.requireAdmin(w, r) {
			return
		}
		user, err := s.userService.Get(r.Context(), userID)
		if err != nil {
			if errors.Is(err, authdomain.ErrUserNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
			} else {
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"user": user})
	case http.MethodPut, http.MethodPatch:
		if !s.requireAdmin(w, r) {
			return
		}

		var payload struct {
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			if errors.Is(err, io.EOF) {
				writeError(w, http.StatusBadRequest, "role is required")
			} else {
				writeError(w, http.StatusBadRequest, "invalid JSON payload")
			}
			return
		}

		role := strings.TrimSpace(payload.Role)
		if role == "" {
			writeError(w, http.StatusBadRequest, "role is required")
			return
		}

		user, err := s.userService.Update(r.Context(), userID, userusecase.UpdateInput{
			Role: &role,
		})
		if err != nil {
			switch {
			case errors.Is(err, authdomain.ErrUserNotFound):
				writeError(w, http.StatusNotFound, err.Error())
			case errors.Is(err, authdomain.ErrInvalidRole):
				writeError(w, http.StatusBadRequest, err.Error())
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}

		writeJSON(w, http.StatusOK, user)
	case http.MethodDelete:
		if !s.requireAdmin(w, r) {
			return
		}

		defaultRole := string(authdomain.RoleUser)
		user, err := s.userService.Update(r.Context(), userID, userusecase.UpdateInput{
			Role: &defaultRole,
		})
		if err != nil {
			switch {
			case errors.Is(err, authdomain.ErrUserNotFound):
				writeError(w, http.StatusNotFound, err.Error())
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}

		writeJSON(w, http.StatusOK, user)
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

func currentUserFromContext(ctx context.Context) (*authdomain.User, bool) {
	user, ok := ctx.Value(ctxKeyUser{}).(*authdomain.User)
	if !ok || user == nil {
		return nil, false
	}
	return user, true
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	user, ok := currentUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return false
	}
	if user.Role != authdomain.RoleAdmin {
		writeError(w, http.StatusForbidden, "admin privileges required")
		return false
	}
	return true
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
