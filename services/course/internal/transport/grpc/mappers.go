package grpc

import (
	"fmt"

	commonv1 "github.com/Newterios/lms-system-prob/proto/common/v1"
	coursev1 "github.com/Newterios/lms-system-prob/proto/course/v1"
	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/google/uuid"
)

func courseToProto(c *model.Course) *coursev1.Course {
	if c == nil {
		return nil
	}
	return &coursev1.Course{
		Id:          c.ID.String(),
		Title:       c.Title,
		Description: c.Description,
		TeacherId:   c.TeacherID.String(),
		Language:    c.Language,
		CreatedAt:   c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func sectionToProto(s *model.Section) *coursev1.Section {
	if s == nil {
		return nil
	}
	return &coursev1.Section{
		Id:       s.ID.String(),
		CourseId: s.CourseID.String(),
		Title:    s.Title,
		Position: s.Position,
	}
}

func materialToProto(m *model.Material) *coursev1.Material {
	if m == nil {
		return nil
	}
	return &coursev1.Material{
		Id:        m.ID.String(),
		SectionId: m.SectionID.String(),
		Kind:      m.Kind,
		Url:       m.URL,
		Title:     m.Title,
	}
}

func enrollmentToProto(e *model.Enrollment) *coursev1.Enrollment {
	if e == nil {
		return nil
	}
	return &coursev1.Enrollment{
		Id:         e.ID.String(),
		CourseId:   e.CourseID.String(),
		StudentId:  e.StudentID.String(),
		EnrolledAt: e.EnrolledAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func pageInfo(page, pageSize int32, total int64) *commonv1.PageInfo {
	return &commonv1.PageInfo{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}
}

func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid uuid %q: %w", s, model.ErrInvalidInput)
	}
	return id, nil
}

func parseOptUUID(s string) (*uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid %q: %w", s, model.ErrInvalidInput)
	}
	return &id, nil
}

func paginationFrom(p interface{ GetPage() int32; GetPageSize() int32 }) model.Pagination {
	if p == nil {
		return model.Pagination{Page: 1, PageSize: 20}
	}
	page := p.GetPage()
	if page < 1 {
		page = 1
	}
	size := p.GetPageSize()
	if size < 1 {
		size = 20
	}
	return model.Pagination{Page: page, PageSize: size}
}
