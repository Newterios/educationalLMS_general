package jwt

import (
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
)

type Config struct {
	AccessSecret  []byte
	RefreshSecret []byte
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type Signer struct{ cfg Config }

func New(cfg Config) *Signer { return &Signer{cfg: cfg} }

func (s *Signer) SignAccess(userID, sessionID, role string, emailVerified bool) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(s.cfg.AccessTTL)
	claims := gojwt.MapClaims{
		"sub":            userID,
		"jti":            sessionID,
		"role":           role,
		"email_verified": emailVerified,
		"iat":            now.Unix(),
		"exp":            exp.Unix(),
	}
	tok, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString(s.cfg.AccessSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return tok, exp, nil
}

func (s *Signer) SignRefresh(userID, sessionID string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(s.cfg.RefreshTTL)
	claims := gojwt.MapClaims{
		"sub": userID,
		"jti": sessionID,
		"iat": now.Unix(),
		"exp": exp.Unix(),
	}
	tok, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString(s.cfg.RefreshSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign refresh token: %w", err)
	}
	return tok, exp, nil
}

func (s *Signer) ParseAccess(token string) (port.AccessClaims, error) {
	parsed, err := gojwt.Parse(token, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.cfg.AccessSecret, nil
	})
	if err != nil || !parsed.Valid {
		return port.AccessClaims{}, model.ErrUnauthenticated
	}
	mc, ok := parsed.Claims.(gojwt.MapClaims)
	if !ok {
		return port.AccessClaims{}, model.ErrUnauthenticated
	}
	ev, _ := mc["email_verified"].(bool)
	return port.AccessClaims{
		UserID:        stringClaim(mc, "sub"),
		SessionID:     stringClaim(mc, "jti"),
		Role:          stringClaim(mc, "role"),
		EmailVerified: ev,
	}, nil
}

func (s *Signer) ParseRefresh(token string) (port.RefreshClaims, error) {
	parsed, err := gojwt.Parse(token, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.cfg.RefreshSecret, nil
	})
	if err != nil || !parsed.Valid {
		return port.RefreshClaims{}, model.ErrUnauthenticated
	}
	mc, ok := parsed.Claims.(gojwt.MapClaims)
	if !ok {
		return port.RefreshClaims{}, model.ErrUnauthenticated
	}
	return port.RefreshClaims{
		UserID:    stringClaim(mc, "sub"),
		SessionID: stringClaim(mc, "jti"),
	}, nil
}

func stringClaim(mc gojwt.MapClaims, key string) string {
	v, _ := mc[key].(string)
	return v
}
