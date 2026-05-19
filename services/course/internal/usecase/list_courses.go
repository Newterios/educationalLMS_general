package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
	"github.com/google/uuid"
)

type ListCoursesUseCase struct {
	courses port.CourseRepository
	cache   port.Cache
}

func NewListCoursesUseCase(courses port.CourseRepository, cache port.Cache) *ListCoursesUseCase {
	return &ListCoursesUseCase{courses: courses, cache: cache}
}

type ListCoursesInput struct {
	TeacherID  *uuid.UUID
	Pagination model.Pagination
}

type ListCoursesOutput struct {
	Courses    []*model.Course
	TotalCount int64
}

type listCoursesCache struct {
	Courses    []*model.Course `json:"courses"`
	TotalCount int64           `json:"total"`
}

func (uc *ListCoursesUseCase) Execute(ctx context.Context, in ListCoursesInput) (ListCoursesOutput, error) {
	key := listCoursesCacheKey(in)

	if raw, err := uc.cache.Get(ctx, key); err == nil && raw != nil {
		var cached listCoursesCache
		if json.Unmarshal(raw, &cached) == nil {
			return ListCoursesOutput{Courses: cached.Courses, TotalCount: cached.TotalCount}, nil
		}
	}

	filter := port.CourseFilter{TeacherID: in.TeacherID}
	courses, total, err := uc.courses.List(ctx, filter, in.Pagination)
	if err != nil {
		return ListCoursesOutput{}, err
	}

	if raw, err := json.Marshal(listCoursesCache{Courses: courses, TotalCount: total}); err == nil {
		_ = uc.cache.Set(ctx, key, raw, courseCacheTTL)
	}

	return ListCoursesOutput{Courses: courses, TotalCount: total}, nil
}

func listCoursesCacheKey(in ListCoursesInput) string {
	tid := ""
	if in.TeacherID != nil {
		tid = in.TeacherID.String()
	}
	return fmt.Sprintf("course:courses:list:%s:%d:%d", tid, in.Pagination.Page, in.Pagination.PageSize)
}
