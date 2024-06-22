package cards

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/pkg/errors"
)

var ErrCardNotFound = fmt.Errorf("card not found")

type Repository interface {
	FindByName(ctx context.Context, name string, page common.Page) (Cards, error)
	FindByNameWithAmount(ctx context.Context, name string, collector Collector, page common.Page) (Cards, error)
}

type postgresRepository struct {
	db  *postgres.DBConnection
	cfg config.Images
}

func NewRepository(connection *postgres.DBConnection, cfg config.Images) Repository {
	return &postgresRepository{
		db:  connection,
		cfg: cfg,
	}
}

// FindByName finds all cards that contain the given name. A matching card face will be separate entry e.g.
// if front and back side of a card match, two entries will be returned.
func (r *postgresRepository) FindByName(ctx context.Context, name string, page common.Page) (Cards, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Empty(page), nil
	}

	query := `
		SELECT
			 face.card_id, face.name,
		     NULLIF(CONCAT($4::text, max(image.image_path)), $4::text)
		FROM
			card_face AS face
		LEFT JOIN
			card_image as image
		ON
			face.card_id = image.card_id
		AND
			(face.id = image.face_id OR image.face_id IS NULL) 
		WHERE
			face.name ILIKE '%' || $1 || '%' AND (image.lang_lang = $5 OR image.lang_lang IS NULL)
		GROUP BY
			face.card_id, face.name
		ORDER BY
			face.name
		LIMIT $2
		OFFSET $3`
	rows, err := r.db.Conn.Query(ctx, query, name, page.Size(), page.Offset(), r.cfg.Host, DefaultLang)
	if err != nil {
		return Empty(page), fmt.Errorf("failed to execute paged card face select %w", err)
	}
	defer rows.Close()

	var result []Card
	for rows.Next() {
		var entry dbCard
		if err = rows.Scan(&entry.CardID, &entry.Name, &entry.Image); err != nil {
			return Empty(page), fmt.Errorf("failed to execute card scan after select %w", err)
		}
		result = append(result, toCard(&entry))
	}
	if rows.Err() != nil {
		return Empty(page), fmt.Errorf("failed to read next row %w", rows.Err())
	}

	return NewCards(result, page), nil
}

// FindByNameWithAmount finds all cards that contain the given name with their quantity in the collection of the
// given collector. A matching card face will be separate entry e.g. if front and back side of a card match,
// two entries will be returned.
func (r *postgresRepository) FindByNameWithAmount(
	ctx context.Context, name string, collector Collector, page common.Page) (Cards, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Empty(page), nil
	}

	query := `
		SELECT
			face.card_id, face.name,
			NULLIF(CONCAT($4::text, max(image.image_path)), $4::text), coalesce(min(card_collection.amount), 0)
		FROM
			card_face AS face
		LEFT JOIN
			card_collection
		ON
			face.card_id = card_collection.card_id
		AND
			card_collection.user_id = $6
		LEFT JOIN
			card_image as image
		ON
			face.card_id = image.card_id
		AND
			(face.id = image.face_id OR image.face_id IS NULL) 
		WHERE
			face.name ILIKE '%' || $1 || '%' AND (image.lang_lang = $5 OR image.lang_lang IS NULL)
		GROUP BY
			face.card_id, face.name
		ORDER BY
			face.name
		LIMIT $2
		OFFSET $3`
	rows, err := r.db.Conn.Query(ctx, query, name, page.Size(), page.Offset(), r.cfg.Host,
		DefaultLang, collector.ID)
	if err != nil {
		return Empty(page), errors.Wrap(err, "failed to execute paged card face select")
	}
	defer rows.Close()

	var result []Card
	for rows.Next() {
		var entry dbCard
		if err = rows.Scan(&entry.CardID, &entry.Name, &entry.Image, &entry.Amount); err != nil {
			return Empty(page), fmt.Errorf("failed to execute card scan after select %w", err)
		}
		result = append(result, toCard(&entry))
	}
	if rows.Err() != nil {
		return Empty(page), fmt.Errorf("failed to read next row %w", rows.Err())
	}

	return NewCards(result, page), nil
}

func offset(p common.Page) int {
	return (p.Page() - 1) * p.Size()
}

type dbCard struct {
	Name   string
	Image  sql.NullString
	CardID int
	Amount int
}

func toCard(dbCard *dbCard) Card {
	return Card{
		ID:     dbCard.CardID,
		Name:   dbCard.Name,
		Image:  dbCard.Image.String,
		Amount: dbCard.Amount,
	}
}
