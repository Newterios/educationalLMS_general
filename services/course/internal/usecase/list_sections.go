package usecase

import (
	"context"
	"encoding/json"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
	"github.com/google/uuid"
)

type ListSectionsUseCase struct {
	sections port.SectionRepository
	cache    port.Cache
}

func NewListSectionsUseCase(sections port.SectionRepository, cache port.Cache) *ListSectionsUseCase {
	return &ListSectionsUseCase{sections: sections, cache: cache}
}

type ListSectionsInput struct{ CourseID uuid.UUID }
type ListSectionsOutput struct{ Sections []*model.Section }

func (uc *ListSectionsUseCase) Execute(ctx context.Context, in ListSectionsInput) (ListSectionsOutput, error) {
	key := "course:sections:" + in.CourseID.String()

	if raw, err := uc.cache.Get(ctx, key); err == nil && raw != nil {
		var sections []*model.Section
		if json.Unmarshal(raw, &sections) == nil {
			return ListSectionsOutput{Sections: sections}, nil
		}
	}

	sections, err := uc.sections.ListByCourseID(ctx, in.CourseID)
	if err != nil {
		return ListSectionsOutput{}, err
	}

	if raw, err := json.Marshal(sections); err == nil {
		_ = uc.cache.Set(ctx, key, raw, courseCacheTTL)
	}

	return ListSectionsOutput{Sections: sections}, nil
}
