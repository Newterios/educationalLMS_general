package jwt

import (
	"fmt"

	gojwt "github.com/golang-jwt/jwt/v5"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
)

type Config struct {
	AccessSecret []byte
}

type Signer struct{ cfg Config }

func New(cfg Config) *Signer { return &Signer{cfg: cfg} }

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

func stringClaim(mc gojwt.MapClaims, key string) string {
	v, _ := mc[key].(string)
	return v
}
