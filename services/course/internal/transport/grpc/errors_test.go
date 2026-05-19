package grpc

import (
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
)

func TestToStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"not found", model.ErrNotFound, codes.NotFound},
		{"already exists", model.ErrAlreadyExists, codes.AlreadyExists},
		{"invalid input", model.ErrInvalidInput, codes.InvalidArgument},
		{"unauthenticated", model.ErrUnauthenticated, codes.Unauthenticated},
		{"permission denied", model.ErrPermissionDenied, codes.PermissionDenied},
		{"failed precondition", model.ErrFailedPrecondition, codes.FailedPrecondition},
		{"rate limited", model.ErrRateLimited, codes.ResourceExhausted},
		{"remote unavailable", model.ErrRemoteUnavailable, codes.Unavailable},
		{"unknown → internal", fmt.Errorf("some unexpected error"), codes.Internal},
		{"wrapped not found", fmt.Errorf("wrapping: %w", model.ErrNotFound), codes.NotFound},
		{"wrapped permission denied", fmt.Errorf("outer: %w", model.ErrPermissionDenied), codes.PermissionDenied},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toStatus(tt.err)
			s, ok := status.FromError(got)
			if !ok {
				t.Fatalf("toStatus(%v) returned non-status error: %v", tt.err, got)
			}
			if s.Code() != tt.want {
				t.Errorf("toStatus(%v) = %s, want %s", tt.err, s.Code(), tt.want)
			}
		})
	}
}
