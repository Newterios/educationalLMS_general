package usecase

import (
	"context"
	"encoding/json"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
	"github.com/google/uuid"
)

type GetCourseUseCase struct {
	courses port.CourseRepository
	cache   port.Cache
}

func NewGetCourseUseCase(courses port.CourseRepository, cache port.Cache) *GetCourseUseCase {
	return &GetCourseUseCase{courses: courses, cache: cache}
}

type GetCourseInput struct{ ID uuid.UUID }
type GetCourseOutput struct{ Course *model.Course }

func (uc *GetCourseUseCase) Execute(ctx context.Context, in GetCourseInput) (GetCourseOutput, error) {
	key := "course:course:" + in.ID.String()

	if raw, err := uc.cache.Get(ctx, key); err == nil && raw != nil {
		var c model.Course
		if json.Unmarshal(raw, &c) == nil {
			return GetCourseOutput{Course: &c}, nil
		}
	}

	c, err := uc.courses.GetByID(ctx, in.ID)
	if err != nil {
		return GetCourseOutput{}, err
	}

	if raw, err := json.Marshal(c); err == nil {
		_ = uc.cache.Set(ctx, key, raw, courseCacheTTL)
	}

	return GetCourseOutput{Course: c}, nil
}
