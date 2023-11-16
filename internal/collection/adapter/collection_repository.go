package adapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"

	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
)

type collectionPostgresRepository struct {
	db  *postgres.DBConnection
	cfg config.Images
}

func NewCollectionRepository(connection *postgres.DBConnection, cfg config.Images) domain.CollectionRepository {
	return &collectionPostgresRepository{
		db:  connection,
		cfg: cfg,
	}
}

// FindByName finds all cards that contain the given name in the collection of the given collector.
// A matching card face will be separate entry e.g. if front and back side of a card match, two entries will
// be returned.
func (r *collectionPostgresRepository) FindByName(name string, page domain.Page,
	collector domain.Collector) (domain.PagedResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.NewEmptyResult(page), nil
	}

	ctx := context.TODO()
	query := `
		SELECT
			face.id, face.name,
			NULLIF(CONCAT($4::text, max(image.image_path)), $4::text), coalesce(min(card_collection.amount), 0)
		FROM
			card_face AS face
		JOIN
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
			face.id, face.name
		ORDER BY
			face.name
		LIMIT $2
		OFFSET $3`
	rows, err := r.db.Conn.Query(ctx, query, name, page.Size(), offset(page), imageBasePath(r.cfg),
		defaultLang, collector.ID)
	if err != nil {
		return domain.PagedResult{}, fmt.Errorf("failed to execute paged card face select %w", err)
	}
	defer rows.Close()

	var result []*domain.Card
	for rows.Next() {
		var entry dbCard
		if err = rows.Scan(&entry.FaceID, &entry.Name, &entry.Image, &entry.Amount); err != nil {
			return domain.PagedResult{}, fmt.Errorf("failed to execute card scan after select %w", err)
		}
		result = append(result, toCard(&entry))
	}
	if rows.Err() != nil {
		return domain.PagedResult{}, fmt.Errorf("failed to read next row %w", rows.Err())
	}

	return domain.NewPagedResult(result, page), nil
}

func (r *collectionPostgresRepository) Add(item domain.Item, collector domain.Collector) error {
	ctx := context.TODO()
	query := `
		INSERT INTO
			card_collection (card_id, user_id, amount)
		VALUES 
			($1, $2, $3)
		ON CONFLICT
			(card_id, user_id)
		DO UPDATE SET
			amount = card_collection.amount + excluded.amount`
	if _, err := r.db.Conn.Exec(ctx, query, item.ID, collector.ID, 1); err != nil {
		return err
	}

	return nil
}

func (r *collectionPostgresRepository) Remove(itemID int, collector domain.Collector) error {
	ctx := context.TODO()
	query := `
		INSERT INTO
			card_collection (card_id, user_id, amount)
		VALUES 
			($1, $2, $3)
		ON CONFLICT
			(card_id, user_id)
		DO UPDATE SET
			amount = card_collection.amount - excluded.amount
		WHERE
			card_collection.amount > 0`
	if _, err := r.db.Conn.Exec(ctx, query, itemID, collector.ID, 1); err != nil {
		return err
	}

	return nil
}

func (r *collectionPostgresRepository) Count(itemID int, collector domain.Collector) (int, error) {
	ctx := context.TODO()
	query := `
		SELECT
			amount
		FROM
			card_collection
		WHERE
			card_id = $1 AND user_id = $2`
	row := r.db.Conn.QueryRow(ctx, query, itemID, collector.ID)
	var count int
	if err := row.Scan(&count); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}

		return 0, fmt.Errorf("failed to execute card scan after select %w", err)
	}

	return count, nil
}
