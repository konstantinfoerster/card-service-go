package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/konstantinfoerster/card-service/internal/common/postgres"
	"github.com/konstantinfoerster/card-service/internal/search/domain"
	"strings"
)

type PostgresRepository struct {
	db *postgres.DBConnection
}

func NewRepository(connection *postgres.DBConnection) domain.Repository {
	return &PostgresRepository{
		db: connection,
	}
}

func Offset(page domain.Page) int {
	return page.Page() * page.Size()
}

func (r *PostgresRepository) FindByName(name string, page domain.Page) (domain.PagedResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.PagedResult{}, nil
	}
	ctx := context.TODO()
	query := `
		SELECT
			DISTINCT face.name, image.image_path
		FROM
			card
		LEFT JOIN
			card_face AS face
		ON
			card.id = face.card_id
		LEFT JOIN
			card_image as image
		ON
			face.card_id = image.card_id AND (face.id = image.face_id OR image.face_id IS NULL) 
		WHERE
			face.name ILIKE '%' || $1 || '%'
		ORDER BY
			face.name
		LIMIT $2
		OFFSET $3`
	rows, err := r.db.Conn.Query(ctx, query, name, page.Size(), Offset(page))
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
