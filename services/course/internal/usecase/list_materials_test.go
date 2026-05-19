package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/usecasetest"
	"github.com/google/uuid"
)

// errMaterialRepo always returns the configured error from ListBySectionID.
type errMaterialRepo struct{ err error }

func (r *errMaterialRepo) Create(_ context.Context, _ *model.Material) error { return nil }
func (r *errMaterialRepo) ListBySectionID(_ context.Context, _ uuid.UUID) ([]*model.Material, error) {
	return nil, r.err
}

// realMaterialCache is an in-memory cache that actually persists Set values.
type realMaterialCache struct{ store map[string][]byte }

func newRealMaterialCache() *realMaterialCache {
	return &realMaterialCache{store: make(map[string][]byte)}
}
func (c *realMaterialCache) Get(_ context.Context, key string) ([]byte, error) {
	return c.store[key], nil
}
func (c *realMaterialCache) Set(_ context.Context, key string, val []byte, _ time.Duration) error {
	c.store[key] = val
	return nil
}
func (c *realMaterialCache) Delete(_ context.Context, key string) error {
	delete(c.store, key)
	return nil
}
func (c *realMaterialCache) DeleteByPrefix(_ context.Context, prefix string) error {
	for k := range c.store {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.store, k)
		}
	}
	return nil
}

func TestListMaterials_HappyPath(t *testing.T) {
	materials := usecasetest.NewFakeMaterialRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewListMaterialsUseCase(materials, cache)

	sectionID := uuid.New()
	for _, title := range []string{"Slides", "Video"} {
		if err := materials.Create(context.Background(), &model.Material{
			ID:        uuid.New(),
			SectionID: sectionID,
			Kind:      "link",
			URL:       "https://example.com",
			Title:     title,
		}); err != nil {
			t.Fatal(err)
		}
	}

	out, err := uc.Execute(context.Background(), usecase.ListMaterialsInput{SectionID: sectionID})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Materials) != 2 {
		t.Errorf("expected 2 materials, got %d", len(out.Materials))
	}
}

func TestListMaterials_Empty(t *testing.T) {
	materials := usecasetest.NewFakeMaterialRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewListMaterialsUseCase(materials, cache)

	out, err := uc.Execute(context.Background(), usecase.ListMaterialsInput{SectionID: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Materials) != 0 {
		t.Errorf("expected 0 materials, got %d", len(out.Materials))
	}
}

func TestListMaterials_CacheHit(t *testing.T) {
	sectionID := uuid.New()
	repo := usecasetest.NewFakeMaterialRepository()
	if err := repo.Create(context.Background(), &model.Material{
		ID: uuid.New(), SectionID: sectionID, Kind: "pdf", URL: "https://x.com", Title: "Lecture",
	}); err != nil {
		t.Fatal(err)
	}
	cache := newRealMaterialCache()
	uc := usecase.NewListMaterialsUseCase(repo, cache)

	// First call — populates cache.
	if _, err := uc.Execute(context.Background(), usecase.ListMaterialsInput{SectionID: sectionID}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Second call — repo replaced with error, cache must serve the result.
	uc2 := usecase.NewListMaterialsUseCase(&errMaterialRepo{err: errors.New("should not hit db")}, cache)
	out, err := uc2.Execute(context.Background(), usecase.ListMaterialsInput{SectionID: sectionID})
	if err != nil {
		t.Fatalf("cache hit: unexpected error: %v", err)
	}
	if len(out.Materials) != 1 {
		t.Errorf("cache hit: expected 1 material, got %d", len(out.Materials))
	}
}

func TestListMaterials_RepoError(t *testing.T) {
	repoErr := errors.New("storage unavailable")
	uc := usecase.NewListMaterialsUseCase(&errMaterialRepo{err: repoErr}, usecasetest.NewFakeCache())
	_, err := uc.Execute(context.Background(), usecase.ListMaterialsInput{SectionID: uuid.New()})
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}
