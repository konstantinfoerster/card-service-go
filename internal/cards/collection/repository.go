package collection

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/pkg/errors"
)

type Repository interface {
	ByID(ctx context.Context, id int) (cards.Card, error)
	FindCollectedByName(ctx context.Context, name string, collector cards.Collector, page common.Page) (cards.Cards, error)
	Upsert(ctx context.Context, item Item, collector cards.Collector) error
	Remove(ctx context.Context, item Item, collector cards.Collector) error
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

// byID finds a card by its card ID.
func (r *postgresRepository) ByID(ctx context.Context, id int) (cards.Card, error) {
	query := `
		SELECT
			face.card_id, face.name, NULLIF(CONCAT($2::text, max(image.image_path)), $2::text)
		FROM
			card_face AS face
		LEFT JOIN
			card_image as image
		ON
			face.card_id = image.card_id
		AND
			(face.id = image.face_id OR image.face_id IS NULL) 
		WHERE
			face.card_id = $1 AND (image.lang_lang = $3 OR image.lang_lang IS NULL)
		GROUP BY
			face.card_id, face.name
		ORDER BY
			face.name
		LIMIT 1`
	row := r.db.Conn.QueryRow(ctx, query, id, r.cfg.Host, cards.DefaultLang)

	var entry dbCard
	if err := row.Scan(&entry.CardID, &entry.Name, &entry.Image); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return cards.Card{}, fmt.Errorf("card with id %d not found,  %w", id, cards.ErrCardNotFound)
		}

		return cards.Card{}, fmt.Errorf("failed to execute card scan after select %w", err)
	}

	return toCard(&entry), nil
}

// FindCollectedByName finds all cards that contain the given name in the collection of the given collector.
// A matching card face will be separate entry e.g. if front and back side of a card match, two entries will
// be returned.
func (r *postgresRepository) FindCollectedByName(
	ctx context.Context, name string, collector cards.Collector, page common.Page) (cards.Cards, error) {
	orEmptyName := ""
	if strings.TrimSpace(name) == "" {
		orEmptyName = "OR 1 = 1"
	}

	query := fmt.Sprintf(`
		SELECT
			face.card_id, face.name,
			NULLIF(CONCAT($4::text, max(image.image_path)), $4::text), coalesce(min(card_collection.amount), 0)
		FROM
			card_collection
		LEFT JOIN
			card_face AS face
		ON
			card_collection.user_id = $6
		AND
			face.card_id = card_collection.card_id
		LEFT JOIN
			card_image as image
		ON
			face.card_id = image.card_id
		AND
			(face.id = image.face_id OR image.face_id IS NULL) 
		WHERE
			(face.name ILIKE '%%' || $1 || '%%' %s) AND (image.lang_lang = $5 OR image.lang_lang IS NULL)
		GROUP BY
			face.card_id, face.name
		ORDER BY
			face.name
		LIMIT $2
		OFFSET $3`, orEmptyName)
	rows, err := r.db.Conn.Query(ctx, query, name, page.Size(), page.Offset(), r.cfg.Host,
		cards.DefaultLang, collector.ID)
	if err != nil {
		return cards.Empty(page), fmt.Errorf("failed to execute paged card face select %w", err)
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

func (r *postgresRepository) Upsert(ctx context.Context, item Item, collector cards.Collector) error {
	query := `
		INSERT INTO
			card_collection (card_id, amount, user_id)
		VALUES 
			($1, $2, $3)
		ON CONFLICT
			(card_id, user_id)
		DO UPDATE SET
			amount = excluded.amount`
	if _, err := r.db.Conn.Exec(ctx, query, item.ID, item.Amount, collector.ID); err != nil {
		return err
	}

	return nil
}

func (r *postgresRepository) Remove(ctx context.Context, item Item, collector cards.Collector) error {
	query := `
		DELETE FROM
			card_collection
		WHERE
			card_id = $1
		AND
			user_id = $2`
	if _, err := r.db.Conn.Exec(ctx, query, item.ID, collector.ID); err != nil {
		return err
	}

	return nil
}

type dbCard struct {
	Name   string
	Image  sql.NullString
	CardID int
	Amount int
}

func toCard(dbCard *dbCard) cards.Card {
	return cards.Card{
		ID:     dbCard.CardID,
		Name:   dbCard.Name,
		Image:  dbCard.Image.String,
		Amount: dbCard.Amount,
	}
}
