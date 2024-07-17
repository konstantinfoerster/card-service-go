package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/pkg/errors"
)

type postgresCardRepository struct {
	db  *postgres.DBConnection
	cfg config.Images
}

func NewCardRepository(connection *postgres.DBConnection, cfg config.Images) cards.CardRepository {
	return &postgresCardRepository{
		db:  connection,
		cfg: cfg,
	}
}

// FindByName finds all cards that contain the given name. A matching card face will be separate entry e.g.
// if front and back side of a card match, two entries will be returned.
func (r *postgresCardRepository) FindByName(ctx context.Context, name string, page common.Page) (cards.Cards, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return cards.Empty(page), nil
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
	rows, err := r.db.Conn.Query(ctx, query, name, page.Size(), page.Offset(), r.cfg.Host, cards.DefaultLang)
	if err != nil {
		return cards.Empty(page), fmt.Errorf("failed to execute paged card face select %w", err)
	}
	defer rows.Close()

	var result []cards.Card
	for rows.Next() {
		var entry dbCard
		if err = rows.Scan(&entry.CardID, &entry.Name, &entry.Image); err != nil {
			return cards.Empty(page), fmt.Errorf("failed to execute card scan after select %w", err)
		}
		result = append(result, toCard(&entry))
	}
	if rows.Err() != nil {
		return cards.Empty(page), fmt.Errorf("failed to read next row %w", rows.Err())
	}

	return cards.NewCards(result, page), nil
}

// FindByNameWithAmount finds all cards that contain the given name with their quantity in the collection of the
// given collector. A matching card face will be separate entry e.g. if front and back side of a card match,
// two entries will be returned.
func (r *postgresCardRepository) FindByNameWithAmount(
	ctx context.Context, name string, collector cards.Collector, page common.Page) (cards.Cards, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return cards.Empty(page), nil
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
		cards.DefaultLang, collector.ID)
	if err != nil {
		return cards.Empty(page), errors.Wrap(err, "failed to execute paged card face select")
	}
	defer rows.Close()

	var result []cards.Card
	for rows.Next() {
		var entry dbCard
		if err = rows.Scan(&entry.CardID, &entry.Name, &entry.Image, &entry.Amount); err != nil {
			return cards.Empty(page), fmt.Errorf("failed to execute card scan after select %w", err)
		}
		result = append(result, toCard(&entry))
	}
	if rows.Err() != nil {
		return cards.Empty(page), fmt.Errorf("failed to read next row %w", rows.Err())
	}

	return cards.NewCards(result, page), nil
}
