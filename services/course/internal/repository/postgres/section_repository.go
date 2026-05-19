package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/google/uuid"
)

type sectionRepository struct{ pool *pgxpool.Pool }

func NewSectionRepository(pool *pgxpool.Pool) *sectionRepository {
	return &sectionRepository{pool: pool}
}

func (r *sectionRepository) Create(ctx context.Context, s *model.Section) error {
	_, err := db(ctx, r.pool).Exec(ctx,
		`INSERT INTO sections (id, course_id, title, position) VALUES ($1,$2,$3,$4)`,
		s.ID, s.CourseID, s.Title, s.Position,
	)
	if err != nil {
		return fmt.Errorf("create section: %w", err)
	}
	return nil
}

func (r *sectionRepository) ListByCourseID(ctx context.Context, courseID uuid.UUID) ([]*model.Section, error) {
	rows, err := db(ctx, r.pool).Query(ctx,
		`SELECT id, course_id, title, position FROM sections WHERE course_id=$1 ORDER BY position`,
		courseID,
	)
	if err != nil {
		return nil, fmt.Errorf("list sections: %w", err)
	}
	defer rows.Close()

	var sections []*model.Section
	for rows.Next() {
		var s model.Section
		if err := rows.Scan(&s.ID, &s.CourseID, &s.Title, &s.Position); err != nil {
			return nil, fmt.Errorf("scan section: %w", err)
		}
		sections = append(sections, &s)
	}
	return sections, rows.Err()
}

func (r *sectionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Section, error) {
	row := db(ctx, r.pool).QueryRow(ctx,
		`SELECT id, course_id, title, position FROM sections WHERE id=$1`, id)
	var s model.Section
	err := row.Scan(&s.ID, &s.CourseID, &s.Title, &s.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get section: %w", err)
	}
	return &s, nil
}
