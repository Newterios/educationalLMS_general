package bcrypt

import (
	"fmt"

	goBcrypt "golang.org/x/crypto/bcrypt"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
)

const DefaultCost = 12

type Hasher struct{ cost int }

func New(cost int) *Hasher { return &Hasher{cost: cost} }

func (h *Hasher) Hash(plain string) (string, error) {
	b, err := goBcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(b), nil
}

func (h *Hasher) Compare(hash, plain string) error {
	if err := goBcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		return model.ErrUnauthenticated
	}
	return nil
}
