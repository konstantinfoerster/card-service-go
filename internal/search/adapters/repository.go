package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/konstantinfoerster/card-service/internal/common/postgres"
	"github.com/konstantinfoerster/card-service/internal/config"
	"github.com/konstantinfoerster/card-service/internal/search/domain"
	"strings"
)

type PostgresRepository struct {
	db  *postgres.DBConnection
	cfg config.Images
}

func NewRepository(connection *postgres.DBConnection, cfg config.Images) domain.Repository {
	return &PostgresRepository{
		db:  connection,
		cfg: cfg,
	}
}

func (r *PostgresRepository) FindByName(name string, page domain.Page) (domain.PagedResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.NewEmptyResult(page), nil
	}
	ctx := context.TODO()
	query := `
		SELECT
			face.name, NULLIF(CONCAT($4::text, max(image.image_path)), $4::text)
		FROM
			card_face AS face
		LEFT JOIN
			card_image as image
		ON
			face.card_id = image.card_id AND (face.id = image.face_id OR image.face_id IS NULL) 
		WHERE
			face.name ILIKE '%' || $1 || '%' AND image.lang_lang = 'eng'
		GROUP BY
			face.name
		ORDER BY
			face.name
		LIMIT $2
		OFFSET $3`
	rows, err := r.db.Conn.Query(ctx, query, name, page.Size(), offset(page), imageBasePath(r.cfg))
	if err != nil {
		return domain.PagedResult{}, fmt.Errorf("failed to execute paged card face select %w", err)
	}
	defer rows.Close()

	var result []*domain.Card
	for rows.Next() {
		var entry dbCard
		if err := rows.Scan(&entry.Name, &entry.Image); err != nil {
			return domain.PagedResult{}, fmt.Errorf("failed to execute card scan after select %w", err)
		}
		result = append(result, toCard(&entry))
	}
	if rows.Err() != nil {
		return domain.PagedResult{}, fmt.Errorf("failed to read next row %w", rows.Err())
	}

	total, err := r.countByName(name)
	if err != nil {
		return domain.PagedResult{}, err
	}

	return domain.NewPagedResult(result, total, page), nil
}

func offset(p domain.Page) int {
	return (p.Page() - 1) * p.Size()
}

func imageBasePath(cfg config.Images) string {
	if strings.HasSuffix(cfg.Host, "/") {
		return cfg.Host
	}
	return cfg.Host + "/"
}

func (r *PostgresRepository) countByName(name string) (int, error) {
	ctx := context.TODO()
	query := `
		SELECT
			COUNT(DISTINCT face.name)
		FROM
			card
		LEFT JOIN
			card_face AS face
		ON
			card.id = face.card_id
		WHERE
			face.name ILIKE '%' || $1 || '%'`
	row := r.db.Conn.QueryRow(ctx, query, name)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to execute count by name %w", err)
	}
	return count, nil
}

type dbCard struct {
	Name  string
	Image sql.NullString
}

func toCard(dbCard *dbCard) *domain.Card {
	return &domain.Card{
		Name:  dbCard.Name,
		Image: dbCard.Image.String,
	}
}
