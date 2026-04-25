package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase/port"
	"github.com/google/uuid"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeQuizRepo struct {
	quiz  *model.Quiz
	quizzes []*model.Quiz
	total  int64
	err   error
}

func (f *fakeQuizRepo) Create(_ context.Context, q *model.Quiz) error { f.quiz = q; return f.err }
func (f *fakeQuizRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.Quiz, error) {
	return f.quiz, f.err
}
func (f *fakeQuizRepo) Update(_ context.Context, q *model.Quiz) error { f.quiz = q; return f.err }
func (f *fakeQuizRepo) Delete(_ context.Context, _ uuid.UUID) error   { return f.err }
func (f *fakeQuizRepo) ListByCourseID(_ context.Context, _ uuid.UUID, _ model.Pagination) ([]*model.Quiz, int64, error) {
	if f.quizzes != nil {
		return f.quizzes, f.total, f.err
	}
	if f.quiz != nil {
		return []*model.Quiz{f.quiz}, 1, f.err
	}
	return nil, 0, f.err
}

type fakeAttemptRepo struct {
	attempt  *model.Attempt
	attempts []*model.Attempt
	total    int64
	err      error
	exists   bool
}

func (f *fakeAttemptRepo) Create(_ context.Context, a *model.Attempt) error {
	f.attempt = a
	return f.err
}
func (f *fakeAttemptRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.Attempt, error) {
	return f.attempt, f.err
}
func (f *fakeAttemptRepo) Update(_ context.Context, a *model.Attempt) error {
	f.attempt = a
	return f.err
}
func (f *fakeAttemptRepo) List(_ context.Context, _ port.AttemptFilter, _ model.Pagination) ([]*model.Attempt, int64, error) {
	return f.attempts, f.total, f.err
}
func (f *fakeAttemptRepo) ListByCourseID(_ context.Context, _ uuid.UUID) ([]*model.Attempt, error) {
	return f.attempts, f.err
}
func (f *fakeAttemptRepo) ExistsForQuiz(_ context.Context, _ uuid.UUID) (bool, error) {
	return f.exists, f.err
}

type fakeOutboxRepo struct {
	err error
}

func (f *fakeOutboxRepo) Insert(_ context.Context, _ *model.OutboxEntry) error { return f.err }
func (f *fakeOutboxRepo) ListUnpublished(_ context.Context, _ int) ([]*model.OutboxEntry, error) {
	return nil, f.err
}
func (f *fakeOutboxRepo) MarkPublished(_ context.Context, _ int64, _ time.Time) error { return f.err }

type fakeTxManager struct{}

func (f *fakeTxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type fakeEnrollmentChecker struct {
	enrolled bool
	err      error
}

func (f *fakeEnrollmentChecker) IsEnrolled(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return f.enrolled, f.err
}

type fakeEventPublisher struct{}

func (f *fakeEventPublisher) Publish(_ context.Context, _ string, _ []byte) error { return nil }

type noopCacheT struct{}

func (noopCacheT) Get(_ context.Context, _ string) ([]byte, error)                  { return nil, nil }
func (noopCacheT) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error { return nil }
func (noopCacheT) Delete(_ context.Context, _ string) error                         { return nil }

// ── CreateQuiz ────────────────────────────────────────────────────────────────

func TestCreateQuiz_Success(t *testing.T) {
	courseID := uuid.Must(uuid.NewRandom())
	teacherID := uuid.Must(uuid.NewRandom())
	uc := usecase.NewCreateQuizUseCase(&fakeQuizRepo{}, &fakeOutboxRepo{}, &fakeTxManager{})

	out, err := uc.Execute(context.Background(), usecase.CreateQuizInput{
		CourseID: courseID,
		CallerID: teacherID,
		Title:    "Go Basics",
		Questions: []*model.Question{
			{Body: "What is Go?", Points: 1, Choices: []*model.Choice{{Key: "a", Value: "A lang", Correct: true}}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Quiz.TeacherID != teacherID {
		t.Errorf("TeacherID not denormalized: got %v want %v", out.Quiz.TeacherID, teacherID)
	}
	if out.Quiz.CourseID != courseID {
		t.Errorf("CourseID mismatch")
	}
}

func TestCreateQuiz_EmptyTitleFails(t *testing.T) {
	uc := usecase.NewCreateQuizUseCase(&fakeQuizRepo{}, &fakeOutboxRepo{}, &fakeTxManager{})
	_, err := uc.Execute(context.Background(), usecase.CreateQuizInput{
		CourseID: uuid.Must(uuid.NewRandom()),
		CallerID: uuid.Must(uuid.NewRandom()),
		Title:    "   ",
	})
	if !errors.Is(err, model.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

// ── GetQuiz ───────────────────────────────────────────────────────────────────

func TestGetQuiz_ReturnsQuiz(t *testing.T) {
	quiz := &model.Quiz{
		ID:    uuid.Must(uuid.NewRandom()),
		Title: "Test Quiz",
		Questions: []*model.Question{
			{
				ID: uuid.Must(uuid.NewRandom()), Body: "2+2?", Points: 1,
				Choices: []*model.Choice{
					{Key: "a", Value: "Three", Correct: false},
					{Key: "b", Value: "Four", Correct: true},
				},
			},
		},
	}
	uc := usecase.NewGetQuizUseCase(&fakeQuizRepo{quiz: quiz}, noopCacheT{})
	out, err := uc.Execute(context.Background(), usecase.GetQuizInput{ID: quiz.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Quiz.ID != quiz.ID {
		t.Errorf("quiz ID mismatch")
	}
}

func TestGetQuiz_NotFound(t *testing.T) {
	uc := usecase.NewGetQuizUseCase(&fakeQuizRepo{err: model.ErrNotFound}, noopCacheT{})
	_, err := uc.Execute(context.Background(), usecase.GetQuizInput{ID: uuid.Must(uuid.NewRandom())})
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── UpdateQuiz ────────────────────────────────────────────────────────────────

func TestUpdateQuiz_OwnershipCheck(t *testing.T) {
	teacherID := uuid.Must(uuid.NewRandom())
	quiz := &model.Quiz{ID: uuid.Must(uuid.NewRandom()), TeacherID: teacherID, Title: "Old"}
	uc := usecase.NewUpdateQuizUseCase(&fakeQuizRepo{quiz: quiz}, &fakeOutboxRepo{}, noopCacheT{}, &fakeTxManager{})

	// wrong caller
	_, err := uc.Execute(context.Background(), usecase.UpdateQuizInput{
		ID:       quiz.ID,
		CallerID: uuid.Must(uuid.NewRandom()), // different teacher
		Title:    "New",
	})
	if !errors.Is(err, model.ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestUpdateQuiz_Success(t *testing.T) {
	teacherID := uuid.Must(uuid.NewRandom())
	quiz := &model.Quiz{ID: uuid.Must(uuid.NewRandom()), TeacherID: teacherID, Title: "Old"}
	uc := usecase.NewUpdateQuizUseCase(&fakeQuizRepo{quiz: quiz}, &fakeOutboxRepo{}, noopCacheT{}, &fakeTxManager{})

	out, err := uc.Execute(context.Background(), usecase.UpdateQuizInput{
		ID:       quiz.ID,
		CallerID: teacherID,
		Title:    "New Title",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Quiz.Title != "New Title" {
		t.Errorf("title not updated: got %q", out.Quiz.Title)
	}
}

// ── DeleteQuiz ────────────────────────────────────────────────────────────────

func TestDeleteQuiz_WithAttempts_FailedPrecondition(t *testing.T) {
	teacherID := uuid.Must(uuid.NewRandom())
	quiz := &model.Quiz{ID: uuid.Must(uuid.NewRandom()), TeacherID: teacherID}
	uc := usecase.NewDeleteQuizUseCase(
		&fakeQuizRepo{quiz: quiz},
		&fakeAttemptRepo{exists: true},
		&fakeOutboxRepo{}, noopCacheT{}, &fakeTxManager{},
	)
	err := uc.Execute(context.Background(), usecase.DeleteQuizInput{ID: quiz.ID, CallerID: teacherID})
	if !errors.Is(err, model.ErrFailedPrecondition) {
		t.Errorf("expected ErrFailedPrecondition, got %v", err)
	}
}

func TestDeleteQuiz_Success(t *testing.T) {
	teacherID := uuid.Must(uuid.NewRandom())
	quiz := &model.Quiz{ID: uuid.Must(uuid.NewRandom()), TeacherID: teacherID}
	uc := usecase.NewDeleteQuizUseCase(
		&fakeQuizRepo{quiz: quiz},
		&fakeAttemptRepo{exists: false},
		&fakeOutboxRepo{}, noopCacheT{}, &fakeTxManager{},
	)
	if err := uc.Execute(context.Background(), usecase.DeleteQuizInput{ID: quiz.ID, CallerID: teacherID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── ListQuizzes ───────────────────────────────────────────────────────────────

func TestListQuizzes_Success(t *testing.T) {
	quiz := &model.Quiz{ID: uuid.Must(uuid.NewRandom()), Title: "Q1"}
	uc := usecase.NewListQuizzesUseCase(&fakeQuizRepo{quiz: quiz}, noopCacheT{})
	out, err := uc.Execute(context.Background(), usecase.ListQuizzesInput{
		CourseID:   uuid.Must(uuid.NewRandom()),
		Pagination: model.Pagination{Page: 1, PageSize: 10},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Quizzes) != 1 {
		t.Errorf("expected 1 quiz, got %d", len(out.Quizzes))
	}
}

// ── StartAttempt ──────────────────────────────────────────────────────────────

func TestStartAttempt_NotEnrolled(t *testing.T) {
	quiz := &model.Quiz{ID: uuid.Must(uuid.NewRandom()), CourseID: uuid.Must(uuid.NewRandom())}
	uc := usecase.NewStartAttemptUseCase(
		&fakeQuizRepo{quiz: quiz}, &fakeAttemptRepo{},
		&fakeEnrollmentChecker{enrolled: false}, &fakeEventPublisher{},
	)
	_, err := uc.Execute(context.Background(), usecase.StartAttemptInput{
		QuizID:    quiz.ID,
		StudentID: uuid.Must(uuid.NewRandom()),
	})
	if !errors.Is(err, model.ErrFailedPrecondition) {
		t.Errorf("expected ErrFailedPrecondition, got %v", err)
	}
}

func TestStartAttempt_CourseUnavailable(t *testing.T) {
	quiz := &model.Quiz{ID: uuid.Must(uuid.NewRandom()), CourseID: uuid.Must(uuid.NewRandom())}
	uc := usecase.NewStartAttemptUseCase(
		&fakeQuizRepo{quiz: quiz}, &fakeAttemptRepo{},
		&fakeEnrollmentChecker{err: model.ErrRemoteUnavailable}, &fakeEventPublisher{},
	)
	_, err := uc.Execute(context.Background(), usecase.StartAttemptInput{
		QuizID:    quiz.ID,
		StudentID: uuid.Must(uuid.NewRandom()),
	})
	if !errors.Is(err, model.ErrRemoteUnavailable) {
		t.Errorf("expected ErrRemoteUnavailable, got %v", err)
	}
}

func TestStartAttempt_Success(t *testing.T) {
	quiz := &model.Quiz{ID: uuid.Must(uuid.NewRandom()), CourseID: uuid.Must(uuid.NewRandom())}
	uc := usecase.NewStartAttemptUseCase(
		&fakeQuizRepo{quiz: quiz}, &fakeAttemptRepo{},
		&fakeEnrollmentChecker{enrolled: true}, &fakeEventPublisher{},
	)
	out, err := uc.Execute(context.Background(), usecase.StartAttemptInput{
		QuizID:    quiz.ID,
		StudentID: uuid.Must(uuid.NewRandom()),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Attempt.Status != "in_progress" {
		t.Errorf("expected in_progress, got %s", out.Attempt.Status)
	}
}

// ── SubmitAttempt ─────────────────────────────────────────────────────────────

func TestSubmitAttempt_TxRollbackOnOutboxFailure(t *testing.T) {
	studentID := uuid.Must(uuid.NewRandom())
	quizID := uuid.Must(uuid.NewRandom())
	attemptID := uuid.Must(uuid.NewRandom())
	now := time.Now().UTC()
	attempt := &model.Attempt{ID: attemptID, QuizID: quizID, StudentID: studentID,
		StartedAt: now, Status: "in_progress", Answers: make(map[uuid.UUID][]string)}
	quiz := &model.Quiz{ID: quizID, Questions: []*model.Question{}}
	outboxErr := errors.New("outbox: DB down")
	uc := usecase.NewSubmitAttemptUseCase(
		&fakeQuizRepo{quiz: quiz}, &fakeAttemptRepo{attempt: attempt},
		&fakeOutboxRepo{err: outboxErr}, &fakeTxManager{},
	)
	_, err := uc.Execute(context.Background(), usecase.SubmitAttemptInput{
		AttemptID: attemptID, StudentID: studentID, Answers: make(map[uuid.UUID][]string),
	})
	if !errors.Is(err, outboxErr) {
		t.Errorf("expected outbox error, got %v", err)
	}
}

func TestSubmitAttempt_AutoGrade50Percent(t *testing.T) {
	studentID := uuid.Must(uuid.NewRandom())
	quizID := uuid.Must(uuid.NewRandom())
	attemptID := uuid.Must(uuid.NewRandom())
	qID1, qID2 := uuid.Must(uuid.NewRandom()), uuid.Must(uuid.NewRandom())
	now := time.Now().UTC()
	attempt := &model.Attempt{ID: attemptID, QuizID: quizID, StudentID: studentID,
		StartedAt: now, Status: "in_progress", Answers: make(map[uuid.UUID][]string)}
	quiz := &model.Quiz{
		ID: quizID,
		Questions: []*model.Question{
			{ID: qID1, Points: 1, Choices: []*model.Choice{{Key: "a", Correct: true}}},
			{ID: qID2, Points: 1, Choices: []*model.Choice{{Key: "b", Correct: true}}},
		},
	}
	uc := usecase.NewSubmitAttemptUseCase(
		&fakeQuizRepo{quiz: quiz}, &fakeAttemptRepo{attempt: attempt},
		&fakeOutboxRepo{}, &fakeTxManager{},
	)
	out, err := uc.Execute(context.Background(), usecase.SubmitAttemptInput{
		AttemptID: attemptID, StudentID: studentID,
		Answers: map[uuid.UUID][]string{qID1: {"a"}, qID2: {"a"}}, // q2 wrong
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Attempt.AutoScore == nil || *out.Attempt.AutoScore != 50.0 {
		t.Errorf("expected 50.0 auto score, got %v", out.Attempt.AutoScore)
	}
}

func TestSubmitAttempt_NotOwner(t *testing.T) {
	quizID := uuid.Must(uuid.NewRandom())
	now := time.Now().UTC()
	attempt := &model.Attempt{ID: uuid.Must(uuid.NewRandom()), QuizID: quizID,
		StudentID: uuid.Must(uuid.NewRandom()), StartedAt: now, Status: "in_progress",
		Answers: make(map[uuid.UUID][]string)}
	quiz := &model.Quiz{ID: quizID, Questions: []*model.Question{}}
	uc := usecase.NewSubmitAttemptUseCase(
		&fakeQuizRepo{quiz: quiz}, &fakeAttemptRepo{attempt: attempt},
		&fakeOutboxRepo{}, &fakeTxManager{},
	)
	_, err := uc.Execute(context.Background(), usecase.SubmitAttemptInput{
		AttemptID: attempt.ID, StudentID: uuid.Must(uuid.NewRandom()), // different student
		Answers: make(map[uuid.UUID][]string),
	})
	if !errors.Is(err, model.ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got %v", err)
	}
}

// ── GetAttempt ────────────────────────────────────────────────────────────────

func TestGetAttempt_StudentOwn(t *testing.T) {
	studentID := uuid.Must(uuid.NewRandom())
	now := time.Now().UTC()
	attempt := &model.Attempt{ID: uuid.Must(uuid.NewRandom()), StudentID: studentID,
		QuizID: uuid.Must(uuid.NewRandom()), StartedAt: now, Status: "in_progress",
		Answers: make(map[uuid.UUID][]string)}
	uc := usecase.NewGetAttemptUseCase(&fakeAttemptRepo{attempt: attempt}, &fakeQuizRepo{})
	out, err := uc.Execute(context.Background(), usecase.GetAttemptInput{
		ID: attempt.ID, CallerID: studentID, Role: "student",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Attempt.ID != attempt.ID {
		t.Error("wrong attempt returned")
	}
}

func TestGetAttempt_StudentOther_Denied(t *testing.T) {
	now := time.Now().UTC()
	attempt := &model.Attempt{ID: uuid.Must(uuid.NewRandom()), StudentID: uuid.Must(uuid.NewRandom()),
		QuizID: uuid.Must(uuid.NewRandom()), StartedAt: now, Status: "in_progress",
		Answers: make(map[uuid.UUID][]string)}
	uc := usecase.NewGetAttemptUseCase(&fakeAttemptRepo{attempt: attempt}, &fakeQuizRepo{})
	_, err := uc.Execute(context.Background(), usecase.GetAttemptInput{
		ID: attempt.ID, CallerID: uuid.Must(uuid.NewRandom()), Role: "student",
	})
	if !errors.Is(err, model.ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got %v", err)
	}
}

// ── ListAttempts ──────────────────────────────────────────────────────────────

func TestListAttempts_Success(t *testing.T) {
	uc := usecase.NewListAttemptsUseCase(&fakeAttemptRepo{
		attempts: []*model.Attempt{{ID: uuid.Must(uuid.NewRandom())}},
		total:    1,
	})
	out, err := uc.Execute(context.Background(), usecase.ListAttemptsInput{
		Pagination: model.Pagination{Page: 1, PageSize: 10},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.TotalCount != 1 {
		t.Errorf("expected 1, got %d", out.TotalCount)
	}
}

// ── GradeSubmission ───────────────────────────────────────────────────────────

func TestGradeSubmission_TxRollbackOnOutboxFailure(t *testing.T) {
	teacherID := uuid.Must(uuid.NewRandom())
	submittedAt := time.Now().UTC()
	attempt := &model.Attempt{
		ID: uuid.Must(uuid.NewRandom()), QuizID: uuid.Must(uuid.NewRandom()),
		StudentID: uuid.Must(uuid.NewRandom()), StartedAt: submittedAt,
		SubmittedAt: &submittedAt, Status: "submitted", Answers: make(map[uuid.UUID][]string),
	}
	quiz := &model.Quiz{ID: attempt.QuizID, TeacherID: teacherID, CourseID: uuid.Must(uuid.NewRandom())}
	outboxErr := errors.New("outbox: lost")
	uc := usecase.NewGradeSubmissionUseCase(
		&fakeAttemptRepo{attempt: attempt}, &fakeQuizRepo{quiz: quiz},
		&fakeOutboxRepo{err: outboxErr}, noopCacheT{}, &fakeTxManager{},
	)
	_, err := uc.Execute(context.Background(), usecase.GradeSubmissionInput{
		AttemptID: attempt.ID, CallerID: teacherID, ManualScore: 75.0,
	})
	if !errors.Is(err, outboxErr) {
		t.Errorf("expected outbox error, got %v", err)
	}
}

func TestGradeSubmission_Success(t *testing.T) {
	teacherID := uuid.Must(uuid.NewRandom())
	submittedAt := time.Now().UTC()
	attempt := &model.Attempt{
		ID: uuid.Must(uuid.NewRandom()), QuizID: uuid.Must(uuid.NewRandom()),
		StudentID: uuid.Must(uuid.NewRandom()), StartedAt: submittedAt,
		SubmittedAt: &submittedAt, Status: "submitted", Answers: make(map[uuid.UUID][]string),
	}
	quiz := &model.Quiz{ID: attempt.QuizID, TeacherID: teacherID, CourseID: uuid.Must(uuid.NewRandom())}
	uc := usecase.NewGradeSubmissionUseCase(
		&fakeAttemptRepo{attempt: attempt}, &fakeQuizRepo{quiz: quiz},
		&fakeOutboxRepo{}, noopCacheT{}, &fakeTxManager{},
	)
	out, err := uc.Execute(context.Background(), usecase.GradeSubmissionInput{
		AttemptID: attempt.ID, CallerID: teacherID, ManualScore: 88.0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Attempt.ManualScore == nil || *out.Attempt.ManualScore != 88.0 {
		t.Errorf("expected manual_score=88.0, got %v", out.Attempt.ManualScore)
	}
	if out.Attempt.Status != "graded" {
		t.Errorf("expected status=graded, got %s", out.Attempt.Status)
	}
}

// ── GetGradebook ──────────────────────────────────────────────────────────────

func TestGetGradebook_AggregatesEntries(t *testing.T) {
	quizID := uuid.Must(uuid.NewRandom())
	score := 75.0
	attempts := []*model.Attempt{
		{ID: uuid.Must(uuid.NewRandom()), QuizID: quizID,
			StudentID: uuid.Must(uuid.NewRandom()), Status: "graded",
			ManualScore: &score, Answers: make(map[uuid.UUID][]string)},
	}
	quiz := &model.Quiz{ID: quizID, Title: "Midterm"}
	uc := usecase.NewGetGradebookUseCase(
		&fakeAttemptRepo{attempts: attempts}, &fakeQuizRepo{quiz: quiz}, noopCacheT{},
	)
	out, err := uc.Execute(context.Background(), usecase.GetGradebookInput{
		CourseID: uuid.Must(uuid.NewRandom()),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(out.Entries))
	}
	if out.Entries[0].Score != 75.0 {
		t.Errorf("expected score=75.0, got %.2f", out.Entries[0].Score)
	}
}

// ── ExportGrades ──────────────────────────────────────────────────────────────

func TestExportGrades_ContainsStudentID(t *testing.T) {
	quizID := uuid.Must(uuid.NewRandom())
	studentID := uuid.Must(uuid.NewRandom())
	score := 60.0
	attempts := []*model.Attempt{
		{ID: uuid.Must(uuid.NewRandom()), QuizID: quizID, StudentID: studentID,
			Status: "submitted", AutoScore: &score, Answers: make(map[uuid.UUID][]string)},
	}
	quiz := &model.Quiz{ID: quizID, Title: "Final"}
	uc := usecase.NewExportGradesUseCase(
		&fakeAttemptRepo{attempts: attempts}, &fakeQuizRepo{quiz: quiz}, noopCacheT{},
	)
	out, err := uc.Execute(context.Background(), usecase.ExportGradesInput{
		CourseID: uuid.Must(uuid.NewRandom()),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	csvStr := string(out.CSV)
	if len(csvStr) == 0 {
		t.Fatal("expected non-empty CSV")
	}
	if !containsStr(csvStr, studentID.String()) {
		t.Errorf("CSV does not contain student_id %s\nCSV:\n%s", studentID, csvStr)
	}
	if out.Filename == "" {
		t.Error("expected non-empty filename")
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsBytes([]byte(s), []byte(sub)))
}

func containsBytes(b, sub []byte) bool {
	for i := 0; i <= len(b)-len(sub); i++ {
		if string(b[i:i+len(sub)]) == string(sub) {
			return true
		}
	}
	return false
}

// ── GetAttempt — additional error/teacher paths ───────────────────────────────

func TestGetAttempt_RepoError(t *testing.T) {
	repoErr := errors.New("db down")
	uc := usecase.NewGetAttemptUseCase(&fakeAttemptRepo{err: repoErr}, &fakeQuizRepo{})
	_, err := uc.Execute(context.Background(), usecase.GetAttemptInput{
		ID: uuid.Must(uuid.NewRandom()), CallerID: uuid.Must(uuid.NewRandom()), Role: "student",
	})
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}

func TestGetAttempt_TeacherOwnsQuiz_Success(t *testing.T) {
	teacherID := uuid.Must(uuid.NewRandom())
	quizID := uuid.Must(uuid.NewRandom())
	now := time.Now().UTC()
	attempt := &model.Attempt{
		ID: uuid.Must(uuid.NewRandom()), StudentID: uuid.Must(uuid.NewRandom()),
		QuizID: quizID, StartedAt: now, Status: "submitted",
		Answers: make(map[uuid.UUID][]string),
	}
	quiz := &model.Quiz{ID: quizID, TeacherID: teacherID, CourseID: uuid.Must(uuid.NewRandom())}
	uc := usecase.NewGetAttemptUseCase(
		&fakeAttemptRepo{attempt: attempt},
		&fakeQuizRepo{quiz: quiz},
	)
	out, err := uc.Execute(context.Background(), usecase.GetAttemptInput{
		ID: attempt.ID, CallerID: teacherID, Role: "teacher",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Attempt.ID != attempt.ID {
		t.Error("wrong attempt returned")
	}
}

func TestGetAttempt_TeacherNotOwner_Denied(t *testing.T) {
	quizID := uuid.Must(uuid.NewRandom())
	now := time.Now().UTC()
	attempt := &model.Attempt{
		ID: uuid.Must(uuid.NewRandom()), StudentID: uuid.Must(uuid.NewRandom()),
		QuizID: quizID, StartedAt: now, Status: "submitted",
		Answers: make(map[uuid.UUID][]string),
	}
	quiz := &model.Quiz{ID: quizID, TeacherID: uuid.Must(uuid.NewRandom()), CourseID: uuid.Must(uuid.NewRandom())}
	uc := usecase.NewGetAttemptUseCase(
		&fakeAttemptRepo{attempt: attempt},
		&fakeQuizRepo{quiz: quiz},
	)
	_, err := uc.Execute(context.Background(), usecase.GetAttemptInput{
		ID: attempt.ID, CallerID: uuid.Must(uuid.NewRandom()), Role: "teacher",
	})
	if !errors.Is(err, model.ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestGetAttempt_TeacherQuizRepoError(t *testing.T) {
	quizErr := errors.New("quiz not found")
	now := time.Now().UTC()
	attempt := &model.Attempt{
		ID: uuid.Must(uuid.NewRandom()), StudentID: uuid.Must(uuid.NewRandom()),
		QuizID: uuid.Must(uuid.NewRandom()), StartedAt: now, Status: "submitted",
		Answers: make(map[uuid.UUID][]string),
	}
	uc := usecase.NewGetAttemptUseCase(
		&fakeAttemptRepo{attempt: attempt},
		&fakeQuizRepo{err: quizErr},
	)
	_, err := uc.Execute(context.Background(), usecase.GetAttemptInput{
		ID: attempt.ID, CallerID: uuid.Must(uuid.NewRandom()), Role: "teacher",
	})
	if !errors.Is(err, quizErr) {
		t.Errorf("expected quiz repo error, got %v", err)
	}
}

// ── ListQuizzes — cache-hit and error paths ───────────────────────────────────

// realCacheT is an in-memory cache that actually stores values (unlike noopCacheT).
type realCacheT struct {
	store map[string][]byte
}

func newRealCache() *realCacheT { return &realCacheT{store: make(map[string][]byte)} }
func (c *realCacheT) Get(_ context.Context, key string) ([]byte, error) {
	return c.store[key], nil
}
func (c *realCacheT) Set(_ context.Context, key string, val []byte, _ time.Duration) error {
	c.store[key] = val
	return nil
}
func (c *realCacheT) Delete(_ context.Context, key string) error {
	delete(c.store, key)
	return nil
}

func TestListQuizzes_CacheHit(t *testing.T) {
	courseID := uuid.Must(uuid.NewRandom())
	quiz := &model.Quiz{ID: uuid.Must(uuid.NewRandom()), Title: "Cached", CourseID: courseID}
	cache := newRealCache()

	// First call populates cache via the usecase.
	uc := usecase.NewListQuizzesUseCase(&fakeQuizRepo{quiz: quiz}, cache)
	_, err := uc.Execute(context.Background(), usecase.ListQuizzesInput{
		CourseID: courseID, Pagination: model.Pagination{Page: 1, PageSize: 10},
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Second call: repo now errors, but cache should serve the result.
	uc2 := usecase.NewListQuizzesUseCase(&fakeQuizRepo{err: errors.New("should not hit db")}, cache)
	out, err := uc2.Execute(context.Background(), usecase.ListQuizzesInput{
		CourseID: courseID, Pagination: model.Pagination{Page: 1, PageSize: 10},
	})
	if err != nil {
		t.Fatalf("cache hit: unexpected error: %v", err)
	}
	if out.TotalCount != 1 {
		t.Errorf("cache hit: expected TotalCount=1, got %d", out.TotalCount)
	}
}

func TestListQuizzes_RepoError(t *testing.T) {
	repoErr := errors.New("db error")
	uc := usecase.NewListQuizzesUseCase(&fakeQuizRepo{err: repoErr}, noopCacheT{})
	_, err := uc.Execute(context.Background(), usecase.ListQuizzesInput{
		CourseID: uuid.Must(uuid.NewRandom()), Pagination: model.Pagination{Page: 1, PageSize: 10},
	})
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}
