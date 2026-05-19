// Package usecasetest provides hand-crafted in-memory fakes for use-case unit tests.
// All fakes are safe to use from a single goroutine (no locking needed in tests).
package usecasetest

import (
	"context"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
	"github.com/google/uuid"
)

// ─── FakeUserRepository ──────────────────────────────────────────────────────

type FakeUserRepository struct {
	users map[string]*model.User // keyed by ID string
	// ForceCreateErr is returned from Create when non-nil.
	ForceCreateErr error
	// ForceGetErr is returned from GetByID / GetByEmail when non-nil.
	ForceGetErr error
}

func NewFakeUserRepository() *FakeUserRepository {
	return &FakeUserRepository{users: make(map[string]*model.User)}
}

func (r *FakeUserRepository) Create(_ context.Context, u *model.User) error {
	if r.ForceCreateErr != nil {
		return r.ForceCreateErr
	}
	for _, existing := range r.users {
		if existing.Email == u.Email {
			return model.ErrAlreadyExists
		}
	}
	cp := *u
	r.users[u.ID.String()] = &cp
	return nil
}

func (r *FakeUserRepository) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if r.ForceGetErr != nil {
		return nil, r.ForceGetErr
	}
	u, ok := r.users[id.String()]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *FakeUserRepository) GetByEmail(_ context.Context, email string) (*model.User, error) {
	if r.ForceGetErr != nil {
		return nil, r.ForceGetErr
	}
	for _, u := range r.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, model.ErrNotFound
}

func (r *FakeUserRepository) Update(_ context.Context, u *model.User) error {
	if _, ok := r.users[u.ID.String()]; !ok {
		return model.ErrNotFound
	}
	cp := *u
	r.users[u.ID.String()] = &cp
	return nil
}

// ─── FakeSessionRepository ───────────────────────────────────────────────────

type FakeSessionRepository struct {
	sessions map[string]*model.Session // keyed by ID string
	byHash   map[string]*model.Session // keyed by RefreshHash
}

func NewFakeSessionRepository() *FakeSessionRepository {
	return &FakeSessionRepository{
		sessions: make(map[string]*model.Session),
		byHash:   make(map[string]*model.Session),
	}
}

func (r *FakeSessionRepository) Create(_ context.Context, s *model.Session) error {
	cp := *s
	r.sessions[s.ID.String()] = &cp
	r.byHash[s.RefreshHash] = &cp
	return nil
}

func (r *FakeSessionRepository) GetByID(_ context.Context, id uuid.UUID) (*model.Session, error) {
	s, ok := r.sessions[id.String()]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *FakeSessionRepository) GetByRefreshHash(_ context.Context, hash string) (*model.Session, error) {
	s, ok := r.byHash[hash]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *FakeSessionRepository) ListActiveForUser(_ context.Context, userID uuid.UUID) ([]*model.Session, error) {
	var out []*model.Session
	for _, s := range r.sessions {
		if s.UserID == userID && s.RevokedAt == nil {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *FakeSessionRepository) Revoke(_ context.Context, id uuid.UUID, revokedAt time.Time) error {
	s, ok := r.sessions[id.String()]
	if !ok {
		return model.ErrNotFound
	}
	t := revokedAt
	s.RevokedAt = &t
	r.byHash[s.RefreshHash] = s
	return nil
}

func (r *FakeSessionRepository) RevokeAllForUser(_ context.Context, userID uuid.UUID, revokedAt time.Time) error {
	for _, s := range r.sessions {
		if s.UserID == userID {
			t := revokedAt
			s.RevokedAt = &t
		}
	}
	return nil
}

func (r *FakeSessionRepository) RevokeAllExcept(_ context.Context, userID, keepID uuid.UUID, revokedAt time.Time) error {
	for _, s := range r.sessions {
		if s.UserID == userID && s.ID != keepID {
			t := revokedAt
			s.RevokedAt = &t
		}
	}
	return nil
}

// ─── FakeVerificationCodeRepository ─────────────────────────────────────────

type FakeVerificationCodeRepository struct {
	codes  map[string]*model.VerificationCode // keyed by ID
	byHash map[string]*model.VerificationCode // keyed by CodeHash
}

func NewFakeVerificationCodeRepository() *FakeVerificationCodeRepository {
	return &FakeVerificationCodeRepository{
		codes:  make(map[string]*model.VerificationCode),
		byHash: make(map[string]*model.VerificationCode),
	}
}

func (r *FakeVerificationCodeRepository) Create(_ context.Context, c *model.VerificationCode) error {
	cp := *c
	r.codes[c.ID.String()] = &cp
	r.byHash[c.CodeHash] = &cp
	return nil
}

func (r *FakeVerificationCodeRepository) GetByCodeHash(_ context.Context, hash string) (*model.VerificationCode, error) {
	c, ok := r.byHash[hash]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *FakeVerificationCodeRepository) MarkUsed(_ context.Context, id uuid.UUID, usedAt time.Time) error {
	c, ok := r.codes[id.String()]
	if !ok {
		return model.ErrNotFound
	}
	t := usedAt
	c.UsedAt = &t
	r.byHash[c.CodeHash] = c
	return nil
}

// ─── FakeMailer ──────────────────────────────────────────────────────────────

type SentEmail struct {
	To, FullName, Code string
}

type FakeMailer struct {
	VerificationEmails  []SentEmail
	PasswordResetEmails []SentEmail
	ChangedEmails       []SentEmail
	ForceErr            error
}

func (m *FakeMailer) SendVerificationEmail(_ context.Context, to, fullName, code string) error {
	if m.ForceErr != nil {
		return m.ForceErr
	}
	m.VerificationEmails = append(m.VerificationEmails, SentEmail{To: to, FullName: fullName, Code: code})
	return nil
}

func (m *FakeMailer) SendPasswordResetEmail(_ context.Context, to, fullName, code string) error {
	if m.ForceErr != nil {
		return m.ForceErr
	}
	m.PasswordResetEmails = append(m.PasswordResetEmails, SentEmail{To: to, FullName: fullName, Code: code})
	return nil
}

func (m *FakeMailer) SendPasswordChangedEmail(_ context.Context, to, fullName string) error {
	if m.ForceErr != nil {
		return m.ForceErr
	}
	m.ChangedEmails = append(m.ChangedEmails, SentEmail{To: to, FullName: fullName})
	return nil
}

// ─── FakeEventPublisher ──────────────────────────────────────────────────────

type PublishedEvent struct {
	Subject string
	Payload []byte
}

type FakeEventPublisher struct {
	Events   []PublishedEvent
	ForceErr error
}

func (p *FakeEventPublisher) Publish(_ context.Context, subject string, payload []byte) error {
	if p.ForceErr != nil {
		return p.ForceErr
	}
	p.Events = append(p.Events, PublishedEvent{Subject: subject, Payload: payload})
	return nil
}

// ─── FakeClock ───────────────────────────────────────────────────────────────

type FakeClock struct {
	Fixed time.Time
}

func NewFakeClock(t time.Time) *FakeClock { return &FakeClock{Fixed: t} }

func (c *FakeClock) Now() time.Time { return c.Fixed }

// ─── FakeCodeGenerator ───────────────────────────────────────────────────────

type FakeCodeGenerator struct {
	Raw      string
	Hash     string
	ForceErr error
}

func NewFakeCodeGenerator(raw, hash string) *FakeCodeGenerator {
	return &FakeCodeGenerator{Raw: raw, Hash: hash}
}

func (g *FakeCodeGenerator) Generate() (string, string, error) {
	if g.ForceErr != nil {
		return "", "", g.ForceErr
	}
	return g.Raw, g.Hash, nil
}

// ─── FakeTokenSigner ─────────────────────────────────────────────────────────

type FakeTokenSigner struct {
	AccessTokenVal  string
	RefreshTokenVal string
	AccessExp       time.Time
	RefreshExp      time.Time

	// ParsedAccess / ParsedRefresh are returned by the corresponding Parse methods.
	ParsedAccess  port.AccessClaims
	ParsedRefresh port.RefreshClaims

	ForceSignErr  error
	ForceParseErr error
}

func (s *FakeTokenSigner) SignAccess(_, _, _ string, _ bool) (string, time.Time, error) {
	if s.ForceSignErr != nil {
		return "", time.Time{}, s.ForceSignErr
	}
	return s.AccessTokenVal, s.AccessExp, nil
}

func (s *FakeTokenSigner) SignRefresh(_, _ string) (string, time.Time, error) {
	if s.ForceSignErr != nil {
		return "", time.Time{}, s.ForceSignErr
	}
	return s.RefreshTokenVal, s.RefreshExp, nil
}

func (s *FakeTokenSigner) ParseAccess(_ string) (port.AccessClaims, error) {
	if s.ForceParseErr != nil {
		return port.AccessClaims{}, s.ForceParseErr
	}
	return s.ParsedAccess, nil
}

func (s *FakeTokenSigner) ParseRefresh(_ string) (port.RefreshClaims, error) {
	if s.ForceParseErr != nil {
		return port.RefreshClaims{}, s.ForceParseErr
	}
	return s.ParsedRefresh, nil
}

// ─── FakePasswordHasher ──────────────────────────────────────────────────────

type FakePasswordHasher struct {
	// HashFn overrides Hash behaviour when set.
	HashFn func(plain string) (string, error)
	// CompareErr is returned from Compare when non-nil (simulates wrong password).
	CompareErr error
}

func NewFakePasswordHasher() *FakePasswordHasher {
	return &FakePasswordHasher{
		HashFn: func(plain string) (string, error) { return "hashed:" + plain, nil },
	}
}

func (h *FakePasswordHasher) Hash(plain string) (string, error) {
	return h.HashFn(plain)
}

func (h *FakePasswordHasher) Compare(hash, plain string) error {
	if h.CompareErr != nil {
		return h.CompareErr
	}
	if hash != "hashed:"+plain {
		return model.ErrUnauthenticated
	}
	return nil
}

// ─── FakeCache ───────────────────────────────────────────────────────────────

type FakeCache struct {
	data     map[string][]byte
	ForceErr error
}

func NewFakeCache() *FakeCache {
	return &FakeCache{data: make(map[string][]byte)}
}

func (c *FakeCache) Get(_ context.Context, key string) ([]byte, error) {
	if c.ForceErr != nil {
		return nil, c.ForceErr
	}
	v, ok := c.data[key]
	if !ok {
		return nil, nil // cache miss
	}
	return v, nil
}

func (c *FakeCache) Set(_ context.Context, key string, val []byte, _ time.Duration) error {
	if c.ForceErr != nil {
		return c.ForceErr
	}
	c.data[key] = val
	return nil
}

func (c *FakeCache) Delete(_ context.Context, key string) error {
	if c.ForceErr != nil {
		return c.ForceErr
	}
	delete(c.data, key)
	return nil
}

// ─── FakeTxRunner ────────────────────────────────────────────────────────────

// FakeTxRunner executes fn inline with no real DB transaction.
type FakeTxRunner struct{}

func (r *FakeTxRunner) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
