package grpc

import (
	"context"
	"log/slog"

	coursev1 "github.com/Newterios/lms-system-prob/proto/course/v1"
	"github.com/Newterios/lms-system-prob/services/course/internal/transport/grpc/interceptors"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase"
)

type Server struct {
	coursev1.UnimplementedCourseServiceServer

	createCourse    *usecase.CreateCourseUseCase
	getCourse       *usecase.GetCourseUseCase
	updateCourse    *usecase.UpdateCourseUseCase
	deleteCourse    *usecase.DeleteCourseUseCase
	listCourses     *usecase.ListCoursesUseCase
	createSection   *usecase.CreateSectionUseCase
	listSections    *usecase.ListSectionsUseCase
	addMaterial     *usecase.AddMaterialUseCase
	listMaterials   *usecase.ListMaterialsUseCase
	enrollStudent   *usecase.EnrollStudentUseCase
	unenrollStudent *usecase.UnenrollStudentUseCase
	listEnrollments *usecase.ListEnrollmentsUseCase
}

func NewServer(
	createCourse *usecase.CreateCourseUseCase,
	getCourse *usecase.GetCourseUseCase,
	updateCourse *usecase.UpdateCourseUseCase,
	deleteCourse *usecase.DeleteCourseUseCase,
	listCourses *usecase.ListCoursesUseCase,
	createSection *usecase.CreateSectionUseCase,
	listSections *usecase.ListSectionsUseCase,
	addMaterial *usecase.AddMaterialUseCase,
	listMaterials *usecase.ListMaterialsUseCase,
	enrollStudent *usecase.EnrollStudentUseCase,
	unenrollStudent *usecase.UnenrollStudentUseCase,
	listEnrollments *usecase.ListEnrollmentsUseCase,
) *Server {
	return &Server{
		createCourse:    createCourse,
		getCourse:       getCourse,
		updateCourse:    updateCourse,
		deleteCourse:    deleteCourse,
		listCourses:     listCourses,
		createSection:   createSection,
		listSections:    listSections,
		addMaterial:     addMaterial,
		listMaterials:   listMaterials,
		enrollStudent:   enrollStudent,
		unenrollStudent: unenrollStudent,
		listEnrollments: listEnrollments,
	}
}

// ── 1. CreateCourse ───────────────────────────────────────────────────────────

func (s *Server) CreateCourse(ctx context.Context, req *coursev1.CreateCourseRequest) (*coursev1.CreateCourseResponse, error) {
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.createCourse.Execute(ctx, usecase.CreateCourseInput{
		TeacherID:   callerID,
		Title:       req.Title,
		Description: req.Description,
		Language:    req.Language,
	})
	if err != nil {
		slog.WarnContext(ctx, "CreateCourse failed", "err", err)
		return nil, toStatus(err)
	}
	return &coursev1.CreateCourseResponse{Course: courseToProto(out.Course)}, nil
}

// ── 2. GetCourse ──────────────────────────────────────────────────────────────

func (s *Server) GetCourse(ctx context.Context, req *coursev1.GetCourseRequest) (*coursev1.GetCourseResponse, error) {
	id, err := parseUUID(req.Id)
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.getCourse.Execute(ctx, usecase.GetCourseInput{ID: id})
	if err != nil {
		return nil, toStatus(err)
	}
	return &coursev1.GetCourseResponse{Course: courseToProto(out.Course)}, nil
}

// ── 3. UpdateCourse ───────────────────────────────────────────────────────────

func (s *Server) UpdateCourse(ctx context.Context, req *coursev1.UpdateCourseRequest) (*coursev1.UpdateCourseResponse, error) {
	id, err := parseUUID(req.Id)
	if err != nil {
		return nil, toStatus(err)
	}
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.updateCourse.Execute(ctx, usecase.UpdateCourseInput{
		ID:          id,
		CallerID:    callerID,
		Title:       req.Title,
		Description: req.Description,
		Language:    req.Language,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &coursev1.UpdateCourseResponse{Course: courseToProto(out.Course)}, nil
}

// ── 4. DeleteCourse ───────────────────────────────────────────────────────────

func (s *Server) DeleteCourse(ctx context.Context, req *coursev1.DeleteCourseRequest) (*coursev1.DeleteCourseResponse, error) {
	id, err := parseUUID(req.Id)
	if err != nil {
		return nil, toStatus(err)
	}
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	if err := s.deleteCourse.Execute(ctx, usecase.DeleteCourseInput{ID: id, CallerID: callerID}); err != nil {
		return nil, toStatus(err)
	}
	return &coursev1.DeleteCourseResponse{}, nil
}

// ── 5. ListCourses ────────────────────────────────────────────────────────────

func (s *Server) ListCourses(ctx context.Context, req *coursev1.ListCoursesRequest) (*coursev1.ListCoursesResponse, error) {
	teacherID, err := parseOptUUID(req.GetTeacherId())
	if err != nil {
		return nil, toStatus(err)
	}
	p := paginationFrom(req.GetPagination())
	out, err := s.listCourses.Execute(ctx, usecase.ListCoursesInput{
		TeacherID:  teacherID,
		Pagination: p,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	courses := make([]*coursev1.Course, len(out.Courses))
	for i, c := range out.Courses {
		courses[i] = courseToProto(c)
	}
	return &coursev1.ListCoursesResponse{
		Courses:  courses,
		PageInfo: pageInfo(p.Page, p.PageSize, out.TotalCount),
	}, nil
}

// ── 6. CreateSection ──────────────────────────────────────────────────────────

func (s *Server) CreateSection(ctx context.Context, req *coursev1.CreateSectionRequest) (*coursev1.CreateSectionResponse, error) {
	courseID, err := parseUUID(req.CourseId)
	if err != nil {
		return nil, toStatus(err)
	}
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.createSection.Execute(ctx, usecase.CreateSectionInput{
		CourseID: courseID,
		CallerID: callerID,
		Title:    req.Title,
		Position: req.Position,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &coursev1.CreateSectionResponse{Section: sectionToProto(out.Section)}, nil
}

// ── 7. ListSections ───────────────────────────────────────────────────────────

func (s *Server) ListSections(ctx context.Context, req *coursev1.ListSectionsRequest) (*coursev1.ListSectionsResponse, error) {
	courseID, err := parseUUID(req.CourseId)
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.listSections.Execute(ctx, usecase.ListSectionsInput{CourseID: courseID})
	if err != nil {
		return nil, toStatus(err)
	}
	sections := make([]*coursev1.Section, len(out.Sections))
	for i, sec := range out.Sections {
		sections[i] = sectionToProto(sec)
	}
	return &coursev1.ListSectionsResponse{Sections: sections}, nil
}

// ── 8. AddMaterial ────────────────────────────────────────────────────────────

func (s *Server) AddMaterial(ctx context.Context, req *coursev1.AddMaterialRequest) (*coursev1.AddMaterialResponse, error) {
	sectionID, err := parseUUID(req.SectionId)
	if err != nil {
		return nil, toStatus(err)
	}
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.addMaterial.Execute(ctx, usecase.AddMaterialInput{
		SectionID: sectionID,
		CallerID:  callerID,
		Kind:      req.Kind,
		URL:       req.Url,
		Title:     req.Title,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &coursev1.AddMaterialResponse{Material: materialToProto(out.Material)}, nil
}

// ── 9. ListMaterials ──────────────────────────────────────────────────────────

func (s *Server) ListMaterials(ctx context.Context, req *coursev1.ListMaterialsRequest) (*coursev1.ListMaterialsResponse, error) {
	sectionID, err := parseUUID(req.SectionId)
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.listMaterials.Execute(ctx, usecase.ListMaterialsInput{SectionID: sectionID})
	if err != nil {
		return nil, toStatus(err)
	}
	materials := make([]*coursev1.Material, len(out.Materials))
	for i, m := range out.Materials {
		materials[i] = materialToProto(m)
	}
	return &coursev1.ListMaterialsResponse{Materials: materials}, nil
}

// ── 10. EnrollStudent ────────────────────────────────────────────────────────

func (s *Server) EnrollStudent(ctx context.Context, req *coursev1.EnrollStudentRequest) (*coursev1.EnrollStudentResponse, error) {
	courseID, err := parseUUID(req.CourseId)
	if err != nil {
		return nil, toStatus(err)
	}
	studentID, err := parseUUID(req.StudentId)
	if err != nil {
		return nil, toStatus(err)
	}
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.enrollStudent.Execute(ctx, usecase.EnrollStudentInput{
		CourseID:  courseID,
		StudentID: studentID,
		CallerID:  callerID,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &coursev1.EnrollStudentResponse{Enrollment: enrollmentToProto(out.Enrollment)}, nil
}

// ── 11. UnenrollStudent ──────────────────────────────────────────────────────

func (s *Server) UnenrollStudent(ctx context.Context, req *coursev1.UnenrollStudentRequest) (*coursev1.UnenrollStudentResponse, error) {
	courseID, err := parseUUID(req.CourseId)
	if err != nil {
		return nil, toStatus(err)
	}
	studentID, err := parseUUID(req.StudentId)
	if err != nil {
		return nil, toStatus(err)
	}
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	if err := s.unenrollStudent.Execute(ctx, usecase.UnenrollStudentInput{
		CourseID:  courseID,
		StudentID: studentID,
		CallerID:  callerID,
	}); err != nil {
		return nil, toStatus(err)
	}
	return &coursev1.UnenrollStudentResponse{}, nil
}

// ── 12. ListEnrollments ──────────────────────────────────────────────────────

func (s *Server) ListEnrollments(ctx context.Context, req *coursev1.ListEnrollmentsRequest) (*coursev1.ListEnrollmentsResponse, error) {
	courseID, err := parseOptUUID(req.GetCourseId())
	if err != nil {
		return nil, toStatus(err)
	}
	studentID, err := parseOptUUID(req.GetStudentId())
	if err != nil {
		return nil, toStatus(err)
	}
	p := paginationFrom(req.GetPagination())
	out, err := s.listEnrollments.Execute(ctx, usecase.ListEnrollmentsInput{
		CourseID:   courseID,
		StudentID:  studentID,
		Pagination: p,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	enrollments := make([]*coursev1.Enrollment, len(out.Enrollments))
	for i, e := range out.Enrollments {
		enrollments[i] = enrollmentToProto(e)
	}
	return &coursev1.ListEnrollmentsResponse{
		Enrollments: enrollments,
		PageInfo:    pageInfo(p.Page, p.PageSize, out.TotalCount),
	}, nil
}
