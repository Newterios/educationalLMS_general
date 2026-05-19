package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/google/uuid"
)

type materialRepository struct{ pool *pgxpool.Pool }

func NewMaterialRepository(pool *pgxpool.Pool) *materialRepository {
	return &materialRepository{pool: pool}
}

func (r *materialRepository) Create(ctx context.Context, m *model.Material) error {
	_, err := db(ctx, r.pool).Exec(ctx,
		`INSERT INTO materials (id, section_id, kind, url, title) VALUES ($1,$2,$3,$4,$5)`,
		m.ID, m.SectionID, m.Kind, m.URL, m.Title,
	)
	if err != nil {
		return fmt.Errorf("create material: %w", err)
	}
	return nil
}

func (r *materialRepository) ListBySectionID(ctx context.Context, sectionID uuid.UUID) ([]*model.Material, error) {
	rows, err := db(ctx, r.pool).Query(ctx,
		`SELECT id, section_id, kind, url, title FROM materials WHERE section_id=$1`,
		sectionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list materials: %w", err)
	}
	defer rows.Close()

	var materials []*model.Material
	for rows.Next() {
		var m model.Material
		if err := rows.Scan(&m.ID, &m.SectionID, &m.Kind, &m.URL, &m.Title); err != nil {
			return nil, fmt.Errorf("scan material: %w", err)
		}
		materials = append(materials, &m)
	}
	return materials, rows.Err()
}
