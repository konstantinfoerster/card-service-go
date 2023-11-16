package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
)

const defaultLang = "eng"

type cardPostgresRepository struct {
	db  *postgres.DBConnection
	cfg config.Images
}

func NewCardRepository(connection *postgres.DBConnection, cfg config.Images) domain.CardRepository {
	return &cardPostgresRepository{
		db:  connection,
		cfg: cfg,
	}
}

// ByID finds a card by ID.
func (r *cardPostgresRepository) ByID(id int) (*domain.Card, error) {
	ctx := context.TODO()
	query := `
		SELECT
			face.id, face.name, NULLIF(CONCAT($2::text, max(image.image_path)), $2::text)
		FROM
			card_face AS face
		LEFT JOIN
			card_image as image
		ON
			face.card_id = image.card_id
		AND
			(face.id = image.face_id OR image.face_id IS NULL) 
		WHERE
			face.id = $1 AND (image.lang_lang = $3 OR image.lang_lang IS NULL)
		GROUP BY
			face.id, face.name
		ORDER BY
			face.name
		LIMIT 1`
	row := r.db.Conn.QueryRow(ctx, query, id, imageBasePath(r.cfg), defaultLang)

	var entry dbCard
	if err := row.Scan(&entry.FaceID, &entry.Name, &entry.Image); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("card with id %d not found,  %w", id, domain.ErrCardNotFound)
		}

		return nil, fmt.Errorf("failed to execute card scan after select %w", err)
	}

	return toCard(&entry), nil
}

// FindByName finds all cards that contain the given name. A matching card face will be separate entry e.g.
// if front and back side of a card match, two entries will be returned.
func (r *cardPostgresRepository) FindByName(name string, page domain.Page) (domain.PagedResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.NewEmptyResult(page), nil
	}

	ctx := context.TODO()
	query := `
		SELECT
			face.id, face.name, NULLIF(CONCAT($4::text, max(image.image_path)), $4::text)
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
			face.id, face.name
		ORDER BY
			face.name
		LIMIT $2
		OFFSET $3`
	rows, err := r.db.Conn.Query(ctx, query, name, page.Size(), offset(page), imageBasePath(r.cfg), defaultLang)
	if err != nil {
		return domain.PagedResult{}, fmt.Errorf("failed to execute paged card face select %w", err)
	}
	defer rows.Close()

	var result []*domain.Card
	for rows.Next() {
		var entry dbCard
		if err = rows.Scan(&entry.FaceID, &entry.Name, &entry.Image); err != nil {
			return domain.PagedResult{}, fmt.Errorf("failed to execute card scan after select %w", err)
		}
		result = append(result, toCard(&entry))
	}
	if rows.Err() != nil {
		return domain.PagedResult{}, fmt.Errorf("failed to read next row %w", rows.Err())
	}

	return domain.NewPagedResult(result, page), nil
}

// FindByNameAndCollector finds all cards that contain the given name with their quantity in the collection of the
// given collector. A matching card face will be separate entry e.g. if front and back side of a card match,
// two entries will be returned.
func (r *cardPostgresRepository) FindByNameAndCollector(name string, page domain.Page,
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

func offset(p domain.Page) int {
	return (p.Page() - 1) * p.Size()
}

func imageBasePath(cfg config.Images) string {
	if strings.HasSuffix(cfg.Host, "/") {
		return cfg.Host
	}

	return cfg.Host + "/"
}

type dbCard struct {
	FaceID int
	Name   string
	Image  sql.NullString
	Amount int
}

func toCard(dbCard *dbCard) *domain.Card {
	return &domain.Card{
		ID:     dbCard.FaceID,
		Name:   dbCard.Name,
		Image:  dbCard.Image.String,
		Amount: dbCard.Amount,
	}
}