package usecasetest

import (
	"context"
	"sync"
	"time"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
	"github.com/google/uuid"
)

// ── FakeCourseRepository ─────────────────────────────────────────────────────

type FakeCourseRepository struct {
	mu       sync.Mutex
	courses  map[uuid.UUID]*model.Course
	ForceErr error
}

func NewFakeCourseRepository() *FakeCourseRepository {
	return &FakeCourseRepository{courses: make(map[uuid.UUID]*model.Course)}
}

func (r *FakeCourseRepository) Create(_ context.Context, c *model.Course) error {
	if r.ForceErr != nil {
		return r.ForceErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *c
	r.courses[c.ID] = &cp
	return nil
}

func (r *FakeCourseRepository) GetByID(_ context.Context, id uuid.UUID) (*model.Course, error) {
	if r.ForceErr != nil {
		return nil, r.ForceErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.courses[id]
	if !ok || c.DeletedAt != nil {
		return nil, model.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *FakeCourseRepository) Update(_ context.Context, c *model.Course) error {
	if r.ForceErr != nil {
		return r.ForceErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.courses[c.ID]; !ok {
		return model.ErrNotFound
	}
	cp := *c
	r.courses[c.ID] = &cp
	return nil
}

func (r *FakeCourseRepository) SoftDelete(_ context.Context, id uuid.UUID, deletedAt time.Time) error {
	if r.ForceErr != nil {
		return r.ForceErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.courses[id]
	if !ok {
		return model.ErrNotFound
	}
	c.DeletedAt = &deletedAt
	return nil
}

func (r *FakeCourseRepository) List(_ context.Context, filter port.CourseFilter, _ model.Pagination) ([]*model.Course, int64, error) {
	if r.ForceErr != nil {
		return nil, 0, r.ForceErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*model.Course
	for _, c := range r.courses {
		if c.DeletedAt != nil {
			continue
		}
		if filter.TeacherID != nil && c.TeacherID != *filter.TeacherID {
			continue
		}
		cp := *c
		out = append(out, &cp)
	}
	return out, int64(len(out)), nil
}

// ── FakeSectionRepository ─────────────────────────────────────────────────────

type FakeSectionRepository struct {
	mu       sync.Mutex
	sections map[uuid.UUID]*model.Section
}

func NewFakeSectionRepository() *FakeSectionRepository {
	return &FakeSectionRepository{sections: make(map[uuid.UUID]*model.Section)}
}

func (r *FakeSectionRepository) Create(_ context.Context, s *model.Section) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *s
	r.sections[s.ID] = &cp
	return nil
}

func (r *FakeSectionRepository) GetByID(_ context.Context, id uuid.UUID) (*model.Section, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sections[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *FakeSectionRepository) ListByCourseID(_ context.Context, courseID uuid.UUID) ([]*model.Section, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*model.Section
	for _, s := range r.sections {
		if s.CourseID == courseID {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}

// ── FakeMaterialRepository ────────────────────────────────────────────────────

type FakeMaterialRepository struct {
	mu        sync.Mutex
	materials map[uuid.UUID]*model.Material
}

func NewFakeMaterialRepository() *FakeMaterialRepository {
	return &FakeMaterialRepository{materials: make(map[uuid.UUID]*model.Material)}
}

func (r *FakeMaterialRepository) Create(_ context.Context, m *model.Material) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *m
	r.materials[m.ID] = &cp
	return nil
}

func (r *FakeMaterialRepository) ListBySectionID(_ context.Context, sectionID uuid.UUID) ([]*model.Material, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*model.Material
	for _, m := range r.materials {
		if m.SectionID == sectionID {
			cp := *m
			out = append(out, &cp)
		}
	}
	return out, nil
}

// ── FakeEnrollmentRepository ──────────────────────────────────────────────────

type FakeEnrollmentRepository struct {
	mu          sync.Mutex
	enrollments map[uuid.UUID]*model.Enrollment
	ForceErr    error
}

func NewFakeEnrollmentRepository() *FakeEnrollmentRepository {
	return &FakeEnrollmentRepository{enrollments: make(map[uuid.UUID]*model.Enrollment)}
}

func (r *FakeEnrollmentRepository) Create(_ context.Context, e *model.Enrollment) error {
	if r.ForceErr != nil {
		return r.ForceErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.enrollments {
		if existing.CourseID == e.CourseID && existing.StudentID == e.StudentID {
			return model.ErrAlreadyExists
		}
	}
	cp := *e
	r.enrollments[e.ID] = &cp
	return nil
}

func (r *FakeEnrollmentRepository) Delete(_ context.Context, courseID, studentID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, e := range r.enrollments {
		if e.CourseID == courseID && e.StudentID == studentID {
			delete(r.enrollments, id)
			return nil
		}
	}
	return nil
}

func (r *FakeEnrollmentRepository) List(_ context.Context, filter port.EnrollmentFilter, _ model.Pagination) ([]*model.Enrollment, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*model.Enrollment
	for _, e := range r.enrollments {
		if filter.CourseID != nil && e.CourseID != *filter.CourseID {
			continue
		}
		if filter.StudentID != nil && e.StudentID != *filter.StudentID {
			continue
		}
		cp := *e
		out = append(out, &cp)
	}
	return out, int64(len(out)), nil
}

// ── FakeEventPublisher ────────────────────────────────────────────────────────

type FakeEventPublisher struct {
	mu       sync.Mutex
	Events   []FakeEvent
	ForceErr error
}

type FakeEvent struct {
	Subject string
	Payload []byte
}

func (p *FakeEventPublisher) Publish(_ context.Context, subject string, payload []byte) error {
	if p.ForceErr != nil {
		return p.ForceErr
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Events = append(p.Events, FakeEvent{Subject: subject, Payload: payload})
	return nil
}

func (p *FakeEventPublisher) LastSubject() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.Events) == 0 {
		return ""
	}
	return p.Events[len(p.Events)-1].Subject
}

// ── FakeCache ─────────────────────────────────────────────────────────────────

type FakeCache struct {
	mu    sync.Mutex
	store map[string][]byte
}

func NewFakeCache() *FakeCache {
	return &FakeCache{store: make(map[string][]byte)}
}

func (c *FakeCache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.store[key]
	if !ok {
		return nil, nil
	}
	return v, nil
}

func (c *FakeCache) Set(_ context.Context, key string, val []byte, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = val
	return nil
}

func (c *FakeCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
	return nil
}

func (c *FakeCache) DeleteByPrefix(_ context.Context, prefix string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.store {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.store, k)
		}
	}
	return nil
}

// ── FakeClock ─────────────────────────────────────────────────────────────────

type FakeClock struct{ Fixed time.Time }

func (c *FakeClock) Now() time.Time { return c.Fixed }
