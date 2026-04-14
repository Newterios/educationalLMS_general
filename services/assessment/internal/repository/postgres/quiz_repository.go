package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/google/uuid"
)

type quizRepository struct{ pool *pgxpool.Pool }

func NewQuizRepository(pool *pgxpool.Pool) *quizRepository {
	return &quizRepository{pool: pool}
}

func (r *quizRepository) Create(ctx context.Context, q *model.Quiz) error {
	_, err := db(ctx, r.pool).Exec(ctx, `
		INSERT INTO quizzes (id, course_id, teacher_id, title, time_limit_sec, shuffle, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		q.ID, q.CourseID, q.TeacherID, q.Title, q.TimeLimitSec, q.Shuffle, q.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create quiz: %w", err)
	}

	// insert questions
	for _, qs := range q.Questions {
		choicesJSON, _ := json.Marshal(qs.Choices)
		_, err := db(ctx, r.pool).Exec(ctx, `
			INSERT INTO questions (id, quiz_id, body, choices, points)
			VALUES ($1,$2,$3,$4,$5)`,
			qs.ID, q.ID, qs.Body, choicesJSON, qs.Points,
		)
		if err != nil {
			return fmt.Errorf("create question: %w", err)
		}
	}
	return nil
}

func (r *quizRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Quiz, error) {
	row := db(ctx, r.pool).QueryRow(ctx, `
		SELECT id, course_id, teacher_id, title, time_limit_sec, shuffle, created_at
		FROM quizzes WHERE id = $1`, id)

	q, err := scanQuiz(row)
	if err != nil {
		return nil, err
	}

	// load questions
	rows, err := db(ctx, r.pool).Query(ctx, `
		SELECT id, body, choices, points FROM questions WHERE quiz_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		qs, err := scanQuestion(rows, q.ID)
		if err != nil {
			return nil, err
		}
		q.Questions = append(q.Questions, qs)
	}
	return q, rows.Err()
}

func (r *quizRepository) Update(ctx context.Context, q *model.Quiz) error {
	tag, err := db(ctx, r.pool).Exec(ctx, `
		UPDATE quizzes SET title=$1, time_limit_sec=$2, shuffle=$3 WHERE id=$4`,
		q.Title, q.TimeLimitSec, q.Shuffle, q.ID,
	)
	if err != nil {
		return fmt.Errorf("update quiz: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update quiz %s: %w", q.ID, model.ErrNotFound)
	}
	return nil
}

func (r *quizRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := db(ctx, r.pool).Exec(ctx, `DELETE FROM quizzes WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete quiz: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete quiz %s: %w", id, model.ErrNotFound)
	}
	return nil
}

func (r *quizRepository) ListByCourseID(ctx context.Context, courseID uuid.UUID, p model.Pagination) ([]*model.Quiz, int64, error) {
	page := p.Page
	if page < 1 {
		page = 1
	}
	size := p.PageSize
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	var total int64
	if err := db(ctx, r.pool).QueryRow(ctx,
		`SELECT COUNT(*) FROM quizzes WHERE course_id=$1`, courseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count quizzes: %w", err)
	}

	rows, err := db(ctx, r.pool).Query(ctx, `
		SELECT id, course_id, teacher_id, title, time_limit_sec, shuffle, created_at
		FROM quizzes WHERE course_id=$1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		courseID, size, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list quizzes: %w", err)
	}
	defer rows.Close()

	var quizzes []*model.Quiz
	for rows.Next() {
		q, err := scanQuiz(rows)
		if err != nil {
			return nil, 0, err
		}
		quizzes = append(quizzes, q)
	}
	return quizzes, total, rows.Err()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func scanQuiz(row interface{ Scan(...any) error }) (*model.Quiz, error) {
	var q model.Quiz
	if err := row.Scan(&q.ID, &q.CourseID, &q.TeacherID, &q.Title, &q.TimeLimitSec, &q.Shuffle, &q.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan quiz: %w", err)
	}
	return &q, nil
}

func scanQuestion(row interface{ Scan(...any) error }, quizID uuid.UUID) (*model.Question, error) {
	var q model.Question
	var choicesJSON []byte
	if err := row.Scan(&q.ID, &q.Body, &choicesJSON, &q.Points); err != nil {
		return nil, fmt.Errorf("scan question: %w", err)
	}
	q.QuizID = quizID
	if err := json.Unmarshal(choicesJSON, &q.Choices); err != nil {
		return nil, fmt.Errorf("unmarshal choices: %w", err)
	}
	return &q, nil
}
