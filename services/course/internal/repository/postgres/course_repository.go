package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
	"github.com/google/uuid"
)

type courseRepository struct{ pool *pgxpool.Pool }

func NewCourseRepository(pool *pgxpool.Pool) *courseRepository {
	return &courseRepository{pool: pool}
}

func (r *courseRepository) Create(ctx context.Context, c *model.Course) error {
	_, err := db(ctx, r.pool).Exec(ctx, `
		INSERT INTO courses (id, title, description, teacher_id, language, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		c.ID, c.Title, c.Description, c.TeacherID, c.Language, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("create course: %w", model.ErrAlreadyExists)
		}
		return fmt.Errorf("create course: %w", err)
	}
	return nil
}

func (r *courseRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Course, error) {
	row := db(ctx, r.pool).QueryRow(ctx, `
		SELECT id, title, description, teacher_id, language, created_at, updated_at, deleted_at
		FROM courses WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanCourse(row)
}

func (r *courseRepository) Update(ctx context.Context, c *model.Course) error {
	tag, err := db(ctx, r.pool).Exec(ctx, `
		UPDATE courses
		SET title=$1, description=$2, language=$3, updated_at=$4
		WHERE id=$5 AND deleted_at IS NULL`,
		c.Title, c.Description, c.Language, c.UpdatedAt, c.ID,
	)
	if err != nil {
		return fmt.Errorf("update course: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update course %s: %w", c.ID, model.ErrNotFound)
	}
	return nil
}

func (r *courseRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	tag, err := db(ctx, r.pool).Exec(ctx,
		`UPDATE courses SET deleted_at=$1 WHERE id=$2 AND deleted_at IS NULL`,
		deletedAt, id,
	)
	if err != nil {
		return fmt.Errorf("soft-delete course: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("soft-delete course %s: %w", id, model.ErrNotFound)
	}
	return nil
}

func (r *courseRepository) List(ctx context.Context, filter port.CourseFilter, p model.Pagination) ([]*model.Course, int64, error) {
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
		rows   pgx.Rows
		err    error
		total  int64
	)

	if filter.TeacherID != nil {
		err = db(ctx, r.pool).QueryRow(ctx,
			`SELECT COUNT(*) FROM courses WHERE deleted_at IS NULL AND teacher_id = $1`,
			*filter.TeacherID,
		).Scan(&total)
	} else {
		err = db(ctx, r.pool).QueryRow(ctx,
			`SELECT COUNT(*) FROM courses WHERE deleted_at IS NULL`,
		).Scan(&total)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("count courses: %w", err)
	}

	if filter.TeacherID != nil {
		rows, err = db(ctx, r.pool).Query(ctx, `
			SELECT id, title, description, teacher_id, language, created_at, updated_at, deleted_at
			FROM courses
			WHERE deleted_at IS NULL AND teacher_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3`,
			*filter.TeacherID, pageSize, offset,
		)
	} else {
		rows, err = db(ctx, r.pool).Query(ctx, `
			SELECT id, title, description, teacher_id, language, created_at, updated_at, deleted_at
			FROM courses
			WHERE deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`,
			pageSize, offset,
		)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("list courses: %w", err)
	}
	defer rows.Close()

	var courses []*model.Course
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, 0, err
		}
		courses = append(courses, c)
	}
	return courses, total, rows.Err()
}

func scanCourse(row interface{ Scan(...any) error }) (*model.Course, error) {
	var c model.Course
	err := row.Scan(
		&c.ID, &c.Title, &c.Description, &c.TeacherID, &c.Language,
		&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan course: %w", err)
	}
	return &c, nil
}
