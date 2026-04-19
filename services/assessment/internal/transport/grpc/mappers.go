package grpc

import (
	"fmt"

	commonv1 "github.com/Newterios/lms-system-prob/proto/common/v1"
	assessmentv1 "github.com/Newterios/lms-system-prob/proto/assessment/v1"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase"
	"github.com/google/uuid"
)

// ── Quiz mappers ──────────────────────────────────────────────────────────────

func quizToProto(q *model.Quiz) *assessmentv1.Quiz {
	if q == nil {
		return nil
	}
	questions := make([]*assessmentv1.Question, len(q.Questions))
	for i, qs := range q.Questions {
		questions[i] = questionToProto(qs)
	}
	return &assessmentv1.Quiz{
		Id:           q.ID.String(),
		CourseId:     q.CourseID.String(),
		Title:        q.Title,
		TimeLimitSec: q.TimeLimitSec,
		Shuffle:      q.Shuffle,
		CreatedAt:    q.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Questions:    questions,
	}
}

// questionToProto converts a domain Question to the read-path proto shape.
// DESIGN DECISION #11: correct is deliberately NOT included in the proto Choice
// to prevent answer-key leakage. The strip happens here in the mapper.
func questionToProto(qs *model.Question) *assessmentv1.Question {
	if qs == nil {
		return nil
	}
	choices := make([]*assessmentv1.Choice, len(qs.Choices))
	for i, c := range qs.Choices {
		choices[i] = &assessmentv1.Choice{
			Key:   c.Key,
			Value: c.Value,
			// c.Correct intentionally omitted — not present in proto Choice
		}
	}
	return &assessmentv1.Question{
		Id:      qs.ID.String(),
		Body:    qs.Body,
		Choices: choices,
		Points:  qs.Points,
	}
}

// ── Attempt mappers ───────────────────────────────────────────────────────────

func attemptToProto(a *model.Attempt) *assessmentv1.Attempt {
	if a == nil {
		return nil
	}
	var submittedAt string
	if a.SubmittedAt != nil {
		submittedAt = a.SubmittedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	var autoScore, manualScore float64
	if a.AutoScore != nil {
		autoScore = *a.AutoScore
	}
	if a.ManualScore != nil {
		manualScore = *a.ManualScore
	}

	// answers map → proto repeated Answer
	answers := make([]*assessmentv1.Answer, 0, len(a.Answers))
	for qID, keys := range a.Answers {
		for _, k := range keys {
			answers = append(answers, &assessmentv1.Answer{
				QuestionId: qID.String(),
				ChoiceKey:  k,
			})
		}
	}

	return &assessmentv1.Attempt{
		Id:          a.ID.String(),
		QuizId:      a.QuizID.String(),
		StudentId:   a.StudentID.String(),
		StartedAt:   a.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		SubmittedAt: submittedAt,
		AutoScore:   autoScore,
		ManualScore: manualScore,
		Status:      a.Status,
		Answers:     answers,
	}
}

// ── Gradebook mappers ─────────────────────────────────────────────────────────

func gradebookEntryToProto(e *usecase.GradebookEntry) *assessmentv1.GradebookEntry {
	if e == nil {
		return nil
	}
	return &assessmentv1.GradebookEntry{
		StudentId:  e.StudentID.String(),
		QuizId:     e.QuizID.String(),
		QuizTitle:  e.QuizTitle,
		Score:      e.Score,
		Status:     e.Status,
	}
}

// ── Proto input parsers ───────────────────────────────────────────────────────

// questionInputFromProto converts write-path QuestionInput (with ChoiceInput.correct)
// into the domain Question (which includes Correct for storage).
func questionInputFromProto(qi *assessmentv1.QuestionInput) *model.Question {
	choices := make([]*model.Choice, len(qi.Choices))
	for i, c := range qi.Choices {
		choices[i] = &model.Choice{
			Key:     c.Key,
			Value:   c.Value,
			Correct: c.Correct, // carried on write path, stripped on read path
		}
	}
	return &model.Question{
		Body:    qi.Body,
		Choices: choices,
		Points:  qi.Points,
	}
}

// answersFromProto converts proto repeated Answer → map[QuestionID][]ChoiceKey.
func answersFromProto(protoAnswers []*assessmentv1.Answer) (map[uuid.UUID][]string, error) {
	out := make(map[uuid.UUID][]string)
	for _, a := range protoAnswers {
		qID, err := parseUUID(a.QuestionId)
		if err != nil {
			return nil, err
		}
		out[qID] = append(out[qID], a.ChoiceKey)
	}
	return out, nil
}

// ── Utility ───────────────────────────────────────────────────────────────────

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

func paginationFrom(p interface {
	GetPage() int32
	GetPageSize() int32
}) model.Pagination {
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

func pageInfo(page, pageSize int32, total int64) *commonv1.PageInfo {
	return &commonv1.PageInfo{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}
}
