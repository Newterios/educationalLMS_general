package grpc

import (
	"context"
	"log/slog"

	assessmentv1 "github.com/Newterios/lms-system-prob/proto/assessment/v1"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/transport/grpc/interceptors"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase"
)

// Server implements assessmentv1.AssessmentServiceServer.
type Server struct {
	assessmentv1.UnimplementedAssessmentServiceServer

	createQuiz      *usecase.CreateQuizUseCase
	getQuiz         *usecase.GetQuizUseCase
	updateQuiz      *usecase.UpdateQuizUseCase
	deleteQuiz      *usecase.DeleteQuizUseCase
	listQuizzes     *usecase.ListQuizzesUseCase
	startAttempt    *usecase.StartAttemptUseCase
	submitAttempt   *usecase.SubmitAttemptUseCase
	getAttempt      *usecase.GetAttemptUseCase
	listAttempts    *usecase.ListAttemptsUseCase
	gradeSubmission *usecase.GradeSubmissionUseCase
	getGradebook    *usecase.GetGradebookUseCase
	exportGrades    *usecase.ExportGradesUseCase
}

func NewServer(
	createQuiz *usecase.CreateQuizUseCase,
	getQuiz *usecase.GetQuizUseCase,
	updateQuiz *usecase.UpdateQuizUseCase,
	deleteQuiz *usecase.DeleteQuizUseCase,
	listQuizzes *usecase.ListQuizzesUseCase,
	startAttempt *usecase.StartAttemptUseCase,
	submitAttempt *usecase.SubmitAttemptUseCase,
	getAttempt *usecase.GetAttemptUseCase,
	listAttempts *usecase.ListAttemptsUseCase,
	gradeSubmission *usecase.GradeSubmissionUseCase,
	getGradebook *usecase.GetGradebookUseCase,
	exportGrades *usecase.ExportGradesUseCase,
) *Server {
	return &Server{
		createQuiz:      createQuiz,
		getQuiz:         getQuiz,
		updateQuiz:      updateQuiz,
		deleteQuiz:      deleteQuiz,
		listQuizzes:     listQuizzes,
		startAttempt:    startAttempt,
		submitAttempt:   submitAttempt,
		getAttempt:      getAttempt,
		listAttempts:    listAttempts,
		gradeSubmission: gradeSubmission,
		getGradebook:    getGradebook,
		exportGrades:    exportGrades,
	}
}

// ── 1. CreateQuiz ─────────────────────────────────────────────────────────────

func (s *Server) CreateQuiz(ctx context.Context, req *assessmentv1.CreateQuizRequest) (*assessmentv1.CreateQuizResponse, error) {
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	courseID, err := parseUUID(req.CourseId)
	if err != nil {
		return nil, toStatus(err)
	}

	questions := make([]*model.Question, len(req.Questions))
	for i, qi := range req.Questions {
		questions[i] = questionInputFromProto(qi)
	}

	out, err := s.createQuiz.Execute(ctx, usecase.CreateQuizInput{
		CourseID:     courseID,
		CallerID:     callerID,
		Title:        req.Title,
		TimeLimitSec: req.TimeLimitSec,
		Shuffle:      req.Shuffle,
		Questions:    questions,
	})
	if err != nil {
		slog.WarnContext(ctx, "CreateQuiz failed", "err", err)
		return nil, toStatus(err)
	}
	return &assessmentv1.CreateQuizResponse{Quiz: quizToProto(out.Quiz)}, nil
}

// ── 2. GetQuiz ────────────────────────────────────────────────────────────────

func (s *Server) GetQuiz(ctx context.Context, req *assessmentv1.GetQuizRequest) (*assessmentv1.GetQuizResponse, error) {
	id, err := parseUUID(req.Id)
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.getQuiz.Execute(ctx, usecase.GetQuizInput{ID: id})
	if err != nil {
		return nil, toStatus(err)
	}
	return &assessmentv1.GetQuizResponse{Quiz: quizToProto(out.Quiz)}, nil
}

// ── 3. UpdateQuiz ─────────────────────────────────────────────────────────────

func (s *Server) UpdateQuiz(ctx context.Context, req *assessmentv1.UpdateQuizRequest) (*assessmentv1.UpdateQuizResponse, error) {
	id, err := parseUUID(req.Id)
	if err != nil {
		return nil, toStatus(err)
	}
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.updateQuiz.Execute(ctx, usecase.UpdateQuizInput{
		ID:           id,
		CallerID:     callerID,
		Title:        req.Title,
		TimeLimitSec: req.TimeLimitSec,
		Shuffle:      req.Shuffle,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &assessmentv1.UpdateQuizResponse{Quiz: quizToProto(out.Quiz)}, nil
}

// ── 4. DeleteQuiz ─────────────────────────────────────────────────────────────

func (s *Server) DeleteQuiz(ctx context.Context, req *assessmentv1.DeleteQuizRequest) (*assessmentv1.DeleteQuizResponse, error) {
	id, err := parseUUID(req.Id)
	if err != nil {
		return nil, toStatus(err)
	}
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	if err := s.deleteQuiz.Execute(ctx, usecase.DeleteQuizInput{ID: id, CallerID: callerID}); err != nil {
		return nil, toStatus(err)
	}
	return &assessmentv1.DeleteQuizResponse{}, nil
}

// ── 5. ListQuizzes ────────────────────────────────────────────────────────────

func (s *Server) ListQuizzes(ctx context.Context, req *assessmentv1.ListQuizzesRequest) (*assessmentv1.ListQuizzesResponse, error) {
	courseID, err := parseUUID(req.CourseId)
	if err != nil {
		return nil, toStatus(err)
	}
	p := paginationFrom(req.GetPagination())
	out, err := s.listQuizzes.Execute(ctx, usecase.ListQuizzesInput{CourseID: courseID, Pagination: p})
	if err != nil {
		return nil, toStatus(err)
	}
	quizzes := make([]*assessmentv1.Quiz, len(out.Quizzes))
	for i, q := range out.Quizzes {
		quizzes[i] = quizToProto(q)
	}
	return &assessmentv1.ListQuizzesResponse{
		Quizzes:  quizzes,
		PageInfo: pageInfo(p.Page, p.PageSize, out.TotalCount),
	}, nil
}

// ── 6. StartAttempt ───────────────────────────────────────────────────────────

func (s *Server) StartAttempt(ctx context.Context, req *assessmentv1.StartAttemptRequest) (*assessmentv1.StartAttemptResponse, error) {
	studentID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	quizID, err := parseUUID(req.QuizId)
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.startAttempt.Execute(ctx, usecase.StartAttemptInput{
		QuizID:    quizID,
		StudentID: studentID,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &assessmentv1.StartAttemptResponse{Attempt: attemptToProto(out.Attempt)}, nil
}

// ── 7. SubmitAttempt ──────────────────────────────────────────────────────────

func (s *Server) SubmitAttempt(ctx context.Context, req *assessmentv1.SubmitAttemptRequest) (*assessmentv1.SubmitAttemptResponse, error) {
	studentID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	attemptID, err := parseUUID(req.AttemptId)
	if err != nil {
		return nil, toStatus(err)
	}
	answers, err := answersFromProto(req.Answers)
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.submitAttempt.Execute(ctx, usecase.SubmitAttemptInput{
		AttemptID: attemptID,
		StudentID: studentID,
		Answers:   answers,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &assessmentv1.SubmitAttemptResponse{Attempt: attemptToProto(out.Attempt)}, nil
}

// ── 8. GetAttempt ─────────────────────────────────────────────────────────────

func (s *Server) GetAttempt(ctx context.Context, req *assessmentv1.GetAttemptRequest) (*assessmentv1.GetAttemptResponse, error) {
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	id, err := parseUUID(req.Id)
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.getAttempt.Execute(ctx, usecase.GetAttemptInput{
		ID:       id,
		CallerID: callerID,
		Role:     interceptors.RoleFrom(ctx),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &assessmentv1.GetAttemptResponse{Attempt: attemptToProto(out.Attempt)}, nil
}

// ── 9. ListAttempts ───────────────────────────────────────────────────────────

func (s *Server) ListAttempts(ctx context.Context, req *assessmentv1.ListAttemptsRequest) (*assessmentv1.ListAttemptsResponse, error) {
	quizID, err := parseOptUUID(req.GetQuizId())
	if err != nil {
		return nil, toStatus(err)
	}
	studentID, err := parseOptUUID(req.GetStudentId())
	if err != nil {
		return nil, toStatus(err)
	}
	p := paginationFrom(req.GetPagination())
	out, err := s.listAttempts.Execute(ctx, usecase.ListAttemptsInput{
		QuizID:     quizID,
		StudentID:  studentID,
		Pagination: p,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	attempts := make([]*assessmentv1.Attempt, len(out.Attempts))
	for i, a := range out.Attempts {
		attempts[i] = attemptToProto(a)
	}
	return &assessmentv1.ListAttemptsResponse{
		Attempts: attempts,
		PageInfo: pageInfo(p.Page, p.PageSize, out.TotalCount),
	}, nil
}

// ── 10. GradeSubmission ───────────────────────────────────────────────────────

func (s *Server) GradeSubmission(ctx context.Context, req *assessmentv1.GradeSubmissionRequest) (*assessmentv1.GradeSubmissionResponse, error) {
	callerID, err := parseUUID(interceptors.UserIDFrom(ctx))
	if err != nil {
		return nil, toStatus(err)
	}
	attemptID, err := parseUUID(req.AttemptId)
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.gradeSubmission.Execute(ctx, usecase.GradeSubmissionInput{
		AttemptID:   attemptID,
		CallerID:    callerID,
		ManualScore: req.ManualScore,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &assessmentv1.GradeSubmissionResponse{Attempt: attemptToProto(out.Attempt)}, nil
}

// ── 11. GetGradebook ──────────────────────────────────────────────────────────

func (s *Server) GetGradebook(ctx context.Context, req *assessmentv1.GetGradebookRequest) (*assessmentv1.GetGradebookResponse, error) {
	courseID, err := parseUUID(req.CourseId)
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.getGradebook.Execute(ctx, usecase.GetGradebookInput{CourseID: courseID})
	if err != nil {
		return nil, toStatus(err)
	}
	entries := make([]*assessmentv1.GradebookEntry, len(out.Entries))
	for i, e := range out.Entries {
		entries[i] = gradebookEntryToProto(e)
	}
	return &assessmentv1.GetGradebookResponse{Entries: entries}, nil
}

// ── 12. ExportGrades ──────────────────────────────────────────────────────────

func (s *Server) ExportGrades(ctx context.Context, req *assessmentv1.ExportGradesRequest) (*assessmentv1.ExportGradesResponse, error) {
	courseID, err := parseUUID(req.CourseId)
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := s.exportGrades.Execute(ctx, usecase.ExportGradesInput{CourseID: courseID})
	if err != nil {
		return nil, toStatus(err)
	}
	return &assessmentv1.ExportGradesResponse{Csv: out.CSV, Filename: out.Filename}, nil
}
