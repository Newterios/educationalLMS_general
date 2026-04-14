package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase/port"
	"github.com/google/uuid"
)

type attemptRepository struct{ pool *pgxpool.Pool }

func NewAttemptRepository(pool *pgxpool.Pool) *attemptRepository {
	return &attemptRepository{pool: pool}
}

func (r *attemptRepository) Create(ctx context.Context, a *model.Attempt) error {
	answersJSON, _ := json.Marshal(a.Answers)
	_, err := db(ctx, r.pool).Exec(ctx, `
		INSERT INTO attempts (id, quiz_id, student_id, started_at, status, answers)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		a.ID, a.QuizID, a.StudentID, a.StartedAt, a.Status, answersJSON,
	)
	if err != nil {
		return fmt.Errorf("create attempt: %w", err)
	}
	return nil
}

func (r *attemptRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Attempt, error) {
	row := db(ctx, r.pool).QueryRow(ctx, `
		SELECT id, quiz_id, student_id, started_at, submitted_at,
		       auto_score, manual_score, status, answers
		FROM attempts WHERE id=$1`, id)
	return scanAttempt(row)
}

func (r *attemptRepository) Update(ctx context.Context, a *model.Attempt) error {
	answersJSON, _ := json.Marshal(a.Answers)
	tag, err := db(ctx, r.pool).Exec(ctx, `
		UPDATE attempts
		SET submitted_at=$1, auto_score=$2, manual_score=$3, status=$4, answers=$5
		WHERE id=$6`,
		a.SubmittedAt, a.AutoScore, a.ManualScore, a.Status, answersJSON, a.ID,
	)
	if err != nil {
		return fmt.Errorf("update attempt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update attempt %s: %w", a.ID, model.ErrNotFound)
	}
	return nil
}

func (r *attemptRepository) List(ctx context.Context, filter port.AttemptFilter, p model.Pagination) ([]*model.Attempt, int64, error) {
	page := p.Page
	if page < 1 {
		page = 1
	}
	size := p.PageSize
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	// build WHERE clause
	where := "1=1"
	args := []any{}
	argN := 1
	if filter.QuizID != nil {
		where += fmt.Sprintf(" AND quiz_id=$%d", argN)
		args = append(args, *filter.QuizID)
		argN++
	}
	if filter.StudentID != nil {
		where += fmt.Sprintf(" AND student_id=$%d", argN)
		args = append(args, *filter.StudentID)
		argN++
	}

	var total int64
	countArgs := append([]any{}, args...)
	if err := db(ctx, r.pool).QueryRow(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM attempts WHERE %s`, where),
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count attempts: %w", err)
	}

	args = append(args, size, offset)
	rows, err := db(ctx, r.pool).Query(ctx, fmt.Sprintf(`
		SELECT id, quiz_id, student_id, started_at, submitted_at,
		       auto_score, manual_score, status, answers
		FROM attempts WHERE %s
		ORDER BY started_at DESC LIMIT $%d OFFSET $%d`, where, argN, argN+1),
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list attempts: %w", err)
	}
	defer rows.Close()

	var attempts []*model.Attempt
	for rows.Next() {
		a, err := scanAttempt(rows)
		if err != nil {
			return nil, 0, err
		}
		attempts = append(attempts, a)
	}
	return attempts, total, rows.Err()
}

func (r *attemptRepository) ListByCourseID(ctx context.Context, courseID uuid.UUID) ([]*model.Attempt, error) {
	rows, err := db(ctx, r.pool).Query(ctx, `
		SELECT a.id, a.quiz_id, a.student_id, a.started_at, a.submitted_at,
		       a.auto_score, a.manual_score, a.status, a.answers
		FROM attempts a
		JOIN quizzes q ON q.id = a.quiz_id
		WHERE q.course_id = $1`, courseID)
	if err != nil {
		return nil, fmt.Errorf("list attempts by course: %w", err)
	}
	defer rows.Close()

	var attempts []*model.Attempt
	for rows.Next() {
		a, err := scanAttempt(rows)
		if err != nil {
			return nil, err
		}
		attempts = append(attempts, a)
	}
	return attempts, rows.Err()
}

func (r *attemptRepository) ExistsForQuiz(ctx context.Context, quizID uuid.UUID) (bool, error) {
	var exists bool
	err := db(ctx, r.pool).QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM attempts WHERE quiz_id=$1)`, quizID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("exists for quiz: %w", err)
	}
	return exists, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func scanAttempt(row interface{ Scan(...any) error }) (*model.Attempt, error) {
	var a model.Attempt
	var answersJSON []byte
	err := row.Scan(
		&a.ID, &a.QuizID, &a.StudentID, &a.StartedAt, &a.SubmittedAt,
		&a.AutoScore, &a.ManualScore, &a.Status, &answersJSON,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan attempt: %w", err)
	}
	if err := json.Unmarshal(answersJSON, &a.Answers); err != nil {
		return nil, fmt.Errorf("unmarshal answers: %w", err)
	}
	if a.Answers == nil {
		a.Answers = make(map[uuid.UUID][]string)
	}
	return &a, nil
}
