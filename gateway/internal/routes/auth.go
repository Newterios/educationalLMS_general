package routes

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	authv1 "github.com/Newterios/lms-system-prob/proto/auth/v1"
	"google.golang.org/grpc/metadata"
)

// RegisterAuth mounts /api/v1/auth/* routes onto mux.
func RegisterAuth(mux *http.ServeMux, client authv1.AuthServiceClient) {
	mux.HandleFunc("POST /api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		var req authv1.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		ctx := forwardAuth(r.Context(), r)
		resp, err := client.Register(ctx, &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("POST /api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req authv1.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		ctx := forwardAuth(r.Context(), r)
		resp, err := client.Login(ctx, &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("POST /api/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		var req authv1.RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		ctx := forwardAuth(r.Context(), r)
		resp, err := client.RefreshToken(ctx, &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("POST /api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		var req authv1.LogoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		ctx := forwardAuth(r.Context(), r)
		resp, err := client.Logout(ctx, &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("POST /api/v1/auth/verify-email", func(w http.ResponseWriter, r *http.Request) {
		var req authv1.VerifyEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		ctx := forwardAuth(r.Context(), r)
		resp, err := client.VerifyEmail(ctx, &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("POST /api/v1/auth/password/reset-request", func(w http.ResponseWriter, r *http.Request) {
		var req authv1.RequestPasswordResetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		ctx := forwardAuth(r.Context(), r)
		resp, err := client.RequestPasswordReset(ctx, &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("POST /api/v1/auth/password/reset-confirm", func(w http.ResponseWriter, r *http.Request) {
		var req authv1.ConfirmPasswordResetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		ctx := forwardAuth(r.Context(), r)
		resp, err := client.ConfirmPasswordReset(ctx, &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("POST /api/v1/auth/password/change", func(w http.ResponseWriter, r *http.Request) {
		var req authv1.ChangePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		ctx := forwardAuth(r.Context(), r)
		resp, err := client.ChangePassword(ctx, &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/auth/me", func(w http.ResponseWriter, r *http.Request) {
		ctx := forwardAuth(r.Context(), r)
		resp, err := client.GetMe(ctx, &authv1.GetMeRequest{})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("PATCH /api/v1/auth/me", func(w http.ResponseWriter, r *http.Request) {
		var req authv1.UpdateProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		ctx := forwardAuth(r.Context(), r)
		resp, err := client.UpdateProfile(ctx, &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/auth/sessions", func(w http.ResponseWriter, r *http.Request) {
		ctx := forwardAuth(r.Context(), r)
		resp, err := client.ListSessions(ctx, &authv1.ListSessionsRequest{})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("DELETE /api/v1/auth/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := forwardAuth(r.Context(), r)
		resp, err := client.RevokeSession(ctx, &authv1.RevokeSessionRequest{SessionId: r.PathValue("id")})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	slog.Info("auth routes registered")
}

// forwardAuth copies the Bearer token from HTTP Authorization header into
// outgoing gRPC metadata so downstream services can authenticate the call.
func forwardAuth(ctx context.Context, r *http.Request) context.Context {
	token := r.Header.Get("Authorization")
	if token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", token)
	}
	return ctx
}
