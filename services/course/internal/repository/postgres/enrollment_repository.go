package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
	"github.com/google/uuid"
)

type enrollmentRepository struct{ pool *pgxpool.Pool }

func NewEnrollmentRepository(pool *pgxpool.Pool) *enrollmentRepository {
	return &enrollmentRepository{pool: pool}
}

func (r *enrollmentRepository) Create(ctx context.Context, e *model.Enrollment) error {
	_, err := db(ctx, r.pool).Exec(ctx,
		`INSERT INTO enrollments (id, course_id, student_id, enrolled_at) VALUES ($1,$2,$3,$4)`,
		e.ID, e.CourseID, e.StudentID, e.EnrolledAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("enroll: %w", model.ErrAlreadyExists)
		}
		return fmt.Errorf("create enrollment: %w", err)
	}
	return nil
}

func (r *enrollmentRepository) Delete(ctx context.Context, courseID, studentID uuid.UUID) error {
	_, err := db(ctx, r.pool).Exec(ctx,
		`DELETE FROM enrollments WHERE course_id=$1 AND student_id=$2`,
		courseID, studentID,
	)
	if err != nil {
		return fmt.Errorf("delete enrollment: %w", err)
	}
	return nil
}

func (r *enrollmentRepository) List(ctx context.Context, filter port.EnrollmentFilter, p model.Pagination) ([]*model.Enrollment, int64, error) {
	page := p.Page
	if page < 1 {
		page = 1
	}
	pageSize := p.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var (
		countQuery string
		listQuery  string
		args       []any
	)

	switch {
	case filter.CourseID != nil && filter.StudentID != nil:
		countQuery = `SELECT COUNT(*) FROM enrollments WHERE course_id=$1 AND student_id=$2`
		listQuery = `SELECT id, course_id, student_id, enrolled_at FROM enrollments WHERE course_id=$1 AND student_id=$2 ORDER BY enrolled_at DESC LIMIT $3 OFFSET $4`
		args = []any{*filter.CourseID, *filter.StudentID}
	case filter.CourseID != nil:
		countQuery = `SELECT COUNT(*) FROM enrollments WHERE course_id=$1`
		listQuery = `SELECT id, course_id, student_id, enrolled_at FROM enrollments WHERE course_id=$1 ORDER BY enrolled_at DESC LIMIT $2 OFFSET $3`
		args = []any{*filter.CourseID}
	case filter.StudentID != nil:
		countQuery = `SELECT COUNT(*) FROM enrollments WHERE student_id=$1`
		listQuery = `SELECT id, course_id, student_id, enrolled_at FROM enrollments WHERE student_id=$1 ORDER BY enrolled_at DESC LIMIT $2 OFFSET $3`
		args = []any{*filter.StudentID}
	default:
		countQuery = `SELECT COUNT(*) FROM enrollments`
		listQuery = `SELECT id, course_id, student_id, enrolled_at FROM enrollments ORDER BY enrolled_at DESC LIMIT $1 OFFSET $2`
		args = []any{}
	}

	var total int64
	if err := db(ctx, r.pool).QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count enrollments: %w", err)
	}

	listArgs := append(args, pageSize, offset)
	rows, err := db(ctx, r.pool).Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list enrollments: %w", err)
	}
	defer rows.Close()

	var enrollments []*model.Enrollment
	for rows.Next() {
		var e model.Enrollment
		if err := rows.Scan(&e.ID, &e.CourseID, &e.StudentID, &e.EnrolledAt); err != nil {
			return nil, 0, fmt.Errorf("scan enrollment: %w", err)
		}
		enrollments = append(enrollments, &e)
	}
	return enrollments, total, rows.Err()
}
