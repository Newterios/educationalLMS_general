package usecase

import (
	"context"
	"encoding/json"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
	"github.com/google/uuid"
)

type ListMaterialsUseCase struct {
	materials port.MaterialRepository
	cache     port.Cache
}

func NewListMaterialsUseCase(materials port.MaterialRepository, cache port.Cache) *ListMaterialsUseCase {
	return &ListMaterialsUseCase{materials: materials, cache: cache}
}

type ListMaterialsInput struct{ SectionID uuid.UUID }
type ListMaterialsOutput struct{ Materials []*model.Material }

func (uc *ListMaterialsUseCase) Execute(ctx context.Context, in ListMaterialsInput) (ListMaterialsOutput, error) {
	key := "course:materials:" + in.SectionID.String()

	if raw, err := uc.cache.Get(ctx, key); err == nil && raw != nil {
		var materials []*model.Material
		if json.Unmarshal(raw, &materials) == nil {
			return ListMaterialsOutput{Materials: materials}, nil
		}
	}

	materials, err := uc.materials.ListBySectionID(ctx, in.SectionID)
	if err != nil {
		return ListMaterialsOutput{}, err
	}

	if raw, err := json.Marshal(materials); err == nil {
		_ = uc.cache.Set(ctx, key, raw, courseCacheTTL)
	}

	return ListMaterialsOutput{Materials: materials}, nil
}
