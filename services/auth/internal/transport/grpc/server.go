package grpc

import (
	"context"
	"log/slog"
	"net"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	authv1 "github.com/Newterios/lms-system-prob/proto/auth/v1"
	"github.com/Newterios/lms-system-prob/services/auth/internal/transport/grpc/interceptors"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase"
	"github.com/google/uuid"
)

// Server implements authv1.AuthServiceServer.
// All business logic lives in the use-case layer; this file is pure translation.
type Server struct {
	authv1.UnimplementedAuthServiceServer

	register             *usecase.RegisterUseCase
	login                *usecase.LoginUseCase
	refreshToken         *usecase.RefreshTokenUseCase
	logout               *usecase.LogoutUseCase
	verifyEmail          *usecase.VerifyEmailUseCase
	requestPasswordReset *usecase.RequestPasswordResetUseCase
	confirmPasswordReset *usecase.ConfirmPasswordResetUseCase
	changePassword       *usecase.ChangePasswordUseCase
	getMe                *usecase.GetMeUseCase
	updateProfile        *usecase.UpdateProfileUseCase
	listSessions         *usecase.ListSessionsUseCase
	revokeSession        *usecase.RevokeSessionUseCase
}

func NewServer(
	register *usecase.RegisterUseCase,
	login *usecase.LoginUseCase,
	refreshToken *usecase.RefreshTokenUseCase,
	logout *usecase.LogoutUseCase,
	verifyEmail *usecase.VerifyEmailUseCase,
	requestPasswordReset *usecase.RequestPasswordResetUseCase,
	confirmPasswordReset *usecase.ConfirmPasswordResetUseCase,
	changePassword *usecase.ChangePasswordUseCase,
	getMe *usecase.GetMeUseCase,
	updateProfile *usecase.UpdateProfileUseCase,
	listSessions *usecase.ListSessionsUseCase,
	revokeSession *usecase.RevokeSessionUseCase,
) *Server {
	return &Server{
		register:             register,
		login:                login,
		refreshToken:         refreshToken,
		logout:               logout,
		verifyEmail:          verifyEmail,
		requestPasswordReset: requestPasswordReset,
		confirmPasswordReset: confirmPasswordReset,
		changePassword:       changePassword,
		getMe:                getMe,
		updateProfile:        updateProfile,
		listSessions:         listSessions,
		revokeSession:        revokeSession,
	}
}

// ── 1. Register ───────────────────────────────────────────────────────────────

func (s *Server) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	out, err := s.register.Execute(ctx, usecase.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Locale:   req.Locale,
	})
	if err != nil {
		slog.WarnContext(ctx, "Register failed", "err", err)
		return nil, toStatus(err)
	}
	return &authv1.RegisterResponse{
		UserId:                    out.UserID,
		RequiresEmailVerification: out.RequiresEmailVerification,
	}, nil
}

// ── 2. Login ──────────────────────────────────────────────────────────────────

func (s *Server) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	ua, ip := peerInfo(ctx)
	out, err := s.login.Execute(ctx, usecase.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: ua,
		IP:        ip,
	})
	if err != nil {
		slog.WarnContext(ctx, "Login failed", "err", err)
		return nil, toStatus(err)
	}
	return &authv1.LoginResponse{
		AccessToken:    out.AccessToken,
		RefreshToken:   out.RefreshToken,
		AccessExpiresAt: out.AccessExpiresAt,
	}, nil
}

// ── 3. RefreshToken ───────────────────────────────────────────────────────────

func (s *Server) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	ua, ip := peerInfo(ctx)
	out, err := s.refreshToken.Execute(ctx, usecase.RefreshTokenInput{
		RefreshToken: req.RefreshToken,
		UserAgent:    ua,
		IP:           ip,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &authv1.RefreshTokenResponse{
		AccessToken:    out.AccessToken,
		RefreshToken:   out.RefreshToken,
		AccessExpiresAt: out.AccessExpiresAt,
	}, nil
}

// ── 4. Logout ─────────────────────────────────────────────────────────────────

func (s *Server) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if err := s.logout.Execute(ctx, usecase.LogoutInput{RefreshToken: req.RefreshToken}); err != nil {
		return nil, toStatus(err)
	}
	return &authv1.LogoutResponse{}, nil
}

// ── 5. VerifyEmail ────────────────────────────────────────────────────────────

func (s *Server) VerifyEmail(ctx context.Context, req *authv1.VerifyEmailRequest) (*authv1.VerifyEmailResponse, error) {
	if err := s.verifyEmail.Execute(ctx, usecase.VerifyEmailInput{Code: req.Code}); err != nil {
		return nil, toStatus(err)
	}
	return &authv1.VerifyEmailResponse{}, nil
}

// ── 6. RequestPasswordReset ───────────────────────────────────────────────────

func (s *Server) RequestPasswordReset(ctx context.Context, req *authv1.RequestPasswordResetRequest) (*authv1.RequestPasswordResetResponse, error) {
	if err := s.requestPasswordReset.Execute(ctx, usecase.RequestPasswordResetInput{Email: req.Email}); err != nil {
		return nil, toStatus(err)
	}
	return &authv1.RequestPasswordResetResponse{}, nil
}

// ── 7. ConfirmPasswordReset ───────────────────────────────────────────────────

func (s *Server) ConfirmPasswordReset(ctx context.Context, req *authv1.ConfirmPasswordResetRequest) (*authv1.ConfirmPasswordResetResponse, error) {
	if err := s.confirmPasswordReset.Execute(ctx, usecase.ConfirmPasswordResetInput{
		Code:        req.Code,
		NewPassword: req.NewPassword,
	}); err != nil {
		return nil, toStatus(err)
	}
	return &authv1.ConfirmPasswordResetResponse{}, nil
}

// ── 8. ChangePassword ────────────────────────────────────────────────────────

func (s *Server) ChangePassword(ctx context.Context, req *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error) {
	userID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	sessionID, err := parseUUID(interceptors.SessionIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}

	if err := s.changePassword.Execute(ctx, usecase.ChangePasswordInput{
		UserID:      userID,
		SessionID:   sessionID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}); err != nil {
		return nil, toStatus(err)
	}
	return &authv1.ChangePasswordResponse{}, nil
}

// ── 9. GetMe ─────────────────────────────────────────────────────────────────

func (s *Server) GetMe(ctx context.Context, _ *authv1.GetMeRequest) (*authv1.GetMeResponse, error) {
	userID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}

	out, err := s.getMe.Execute(ctx, usecase.GetMeInput{UserID: userID})
	if err != nil {
		return nil, toStatus(err)
	}
	return &authv1.GetMeResponse{User: userToProto(out.User)}, nil
}

// ── 10. UpdateProfile ────────────────────────────────────────────────────────

func (s *Server) UpdateProfile(ctx context.Context, req *authv1.UpdateProfileRequest) (*authv1.UpdateProfileResponse, error) {
	userID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}

	out, err := s.updateProfile.Execute(ctx, usecase.UpdateProfileInput{
		UserID:   userID,
		FullName: req.FullName,
		Locale:   req.Locale,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &authv1.UpdateProfileResponse{User: userToProto(out.User)}, nil
}

// ── 11. ListSessions ─────────────────────────────────────────────────────────

func (s *Server) ListSessions(ctx context.Context, _ *authv1.ListSessionsRequest) (*authv1.ListSessionsResponse, error) {
	userID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	currentID, _ := uuid.Parse(interceptors.SessionIDFrom(ctx))

	out, err := s.listSessions.Execute(ctx, usecase.ListSessionsInput{
		UserID:    userID,
		CurrentID: currentID,
	})
	if err != nil {
		return nil, toStatus(err)
	}

	sessions := make([]*authv1.Session, len(out.Sessions))
	for i, sess := range out.Sessions {
		sessions[i] = sessionToProto(sess, currentID)
	}
	return &authv1.ListSessionsResponse{Sessions: sessions}, nil
}

// ── 12. RevokeSession ────────────────────────────────────────────────────────

func (s *Server) RevokeSession(ctx context.Context, req *authv1.RevokeSessionRequest) (*authv1.RevokeSessionResponse, error) {
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	sessionID, err := parseUUID(req.SessionId)
	if err != nil {
		return nil, toStatus(err)
	}

	if err := s.revokeSession.Execute(ctx, usecase.RevokeSessionInput{
		CallerID:  callerID,
		SessionID: sessionID,
	}); err != nil {
		return nil, toStatus(err)
	}
	return &authv1.RevokeSessionResponse{}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// peerInfo extracts UserAgent and remote IP from gRPC context metadata/peer.
func peerInfo(ctx context.Context) (userAgent, ip string) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("user-agent"); len(vals) > 0 {
			userAgent = vals[0]
		}
	}
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		// p.Addr.String() is "host:port"; Postgres inet expects just the host.
		host, _, err := net.SplitHostPort(p.Addr.String())
		if err == nil {
			ip = host
		} else {
			ip = p.Addr.String()
		}
	}
	return userAgent, ip
}
