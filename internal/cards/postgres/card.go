package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type PostgresCardRepository struct {
	db  *DBConnection
	cfg Images
}

func NewCollectionRepository(connection *DBConnection, cfg Images) *PostgresCardRepository {
	return &PostgresCardRepository{
		db:  connection,
		cfg: cfg,
	}
}

func NewCardRepository(connection *DBConnection, cfg Images) *PostgresCardRepository {
	return &PostgresCardRepository{
		db:  connection,
		cfg: cfg,
	}
}

func (r *PostgresCardRepository) Find(ctx context.Context, f cards.Filter, page cards.Page) (cards.Cards, error) {
	queryArgs := pgx.NamedArgs{
		"name":    f.Name,
		"limit":   page.Size(),
		"offset":  page.Offset(),
		"baseURL": r.cfg.Host,
		"lang":    f.Lang,
	}

	tplParams := make(map[string]any)
	tplParams["name"] = f.Name
	tplParams["lang"] = f.Lang

	if f.IDs.NotEmpty() {
		cardIDs := make([]int, 0)
		faceIDs := make([]int, 0)
		for _, id := range f.IDs {
			if id.CardID > 0 {
				cardIDs = append(cardIDs, id.CardID)
			}
			if id.FaceID > 0 {
				faceIDs = append(faceIDs, id.FaceID)
			}
		}
		if len(cardIDs) > 0 {
			queryArgs["cardIDs"] = cardIDs
			tplParams["cardIDs"] = cardIDs
		}
		if len(faceIDs) > 0 {
			queryArgs["faceIDs"] = faceIDs
			tplParams["faceIDs"] = faceIDs
		}
	}

	if f.Collector != nil {
		queryArgs["user"] = f.Collector.ID
		tplParams["user"] = f.Collector.ID
		tplParams["onlyCollected"] = f.OnlyCollected
	}

	queryTpl, err := template.New("findcards").Parse(`
WITH
  cte
AS 
(
  SELECT
    DISTINCT ON (face.name)
    row_number() over (partition by face.card_id) as rn,
    face.card_id, face.id, face.name, card.number, set.code, set.name,
    NULLIF(CONCAT(@baseURL::text, image.image_path), @baseURL::text)
    {{if .user}}, coalesce(card_collection.amount, 0) {{else}}, 0 {{end}}
  FROM
    card AS card
  INNER JOIN
    card_set AS set
  ON
    card.card_set_code = set.code
  INNER JOIN
    card_face AS face
  ON
    card.id = face.card_id
  LEFT JOIN
    card_image as image
  ON
    face.id = image.face_id
    {{if .user}}
      {{if .onlyCollected}}
        INNER JOIN
      {{else}}
        LEFT JOIN
      {{end}}
        card_collection
      ON
        face.card_id = card_collection.card_id
      AND
        card_collection.user_id = @user
    {{end}}
  WHERE
    1=1
  {{if .cardIDs}}
    AND
      face.card_id = any(@cardIDs)
  {{end}}
  {{if .faceIDs}}
    AND
      face.id = any(@faceIDs)
  {{end}}
  {{if .name}}
    AND
      (face.name ILIKE '%' || @name || '%')
  {{end}}
  {{if .lang}}
    AND
      (image.lang_lang = @lang OR image.lang_lang IS NULL)
  {{end}}
  ORDER BY
    face.name
)
SELECT 
  * 
FROM 
  cte
WHERE
  rn = 1
LIMIT @limit
OFFSET @offset`)
	if err != nil {
		return cards.Cards{}, fmt.Errorf("failed to parse card find template %w", err)
	}
	var query bytes.Buffer
	if err = queryTpl.Execute(&query, tplParams); err != nil {
		return cards.Cards{}, fmt.Errorf("failed to execute query template %w", err)
	}

	rows, err := r.db.Conn.Query(ctx, query.String(), queryArgs)
	if err != nil {
		return cards.Cards{}, fmt.Errorf("failed to execute paged card face select %w", err)
	}
	defer rows.Close()

	var result []cards.Card
	for rows.Next() {
		var entry dbCard
		err = rows.Scan(
			&entry.Row,
			&entry.CardID,
			&entry.FaceID,
			&entry.Name,
			&entry.Number,
			&entry.SetCode,
			&entry.SetName,
			&entry.ImageURL,
			&entry.Amount,
		)
		if err != nil {
			return cards.Cards{}, fmt.Errorf("failed to execute card scan after select %w", err)
		}
		result = append(result, toCard(entry))
	}
	if rows.Err() != nil {
		return cards.Cards{}, fmt.Errorf("failed to read next row %w", rows.Err())
	}

	return cards.NewCards(result, page), nil
}

func (r *PostgresCardRepository) Prints(
	ctx context.Context, name string, collector cards.Collector, page cards.Page) (cards.CardPrints, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return cards.EmptyCardPrints(page), nil
	}
	queryArgs := pgx.NamedArgs{
		"name":   name,
		"limit":  page.Size(),
		"offset": page.Offset(),
	}

	tplParams := make(map[string]any)
	if collector.ID != "" {
		queryArgs["user"] = collector.ID
		tplParams["user"] = collector.ID
	}

	queryTpl, err := template.New("printcards").Parse(`
SELECT 
  face.name, face.card_id, face.id, card.number, card.card_set_code
  {{if .user}}, coalesce(card_collection.amount, 0) {{else}}, 0 {{end}}
FROM
  card AS card
INNER JOIN
  card_face AS face
ON
  card.id = face.card_id
  {{if .user}}
    LEFT JOIN
      card_collection
	ON
  	  face.card_id = card_collection.card_id
	AND
	  card_collection.user_id = @user
  {{end}}
WHERE
  face.name = @name
ORDER BY
  card.id
LIMIT @limit
OFFSET @offset`)
	if err != nil {
		return cards.CardPrints{}, fmt.Errorf("failed to parse card print template %w", err)
	}

	var query bytes.Buffer
	if err = queryTpl.Execute(&query, tplParams); err != nil {
		return cards.CardPrints{}, fmt.Errorf("failed to execute query template %w", err)
	}

	rows, err := r.db.Conn.Query(ctx, query.String(), queryArgs)
	if err != nil {
		return cards.CardPrints{}, fmt.Errorf("failed to execute paged card print select %w", err)
	}
	defer rows.Close()

	var result []cards.CardPrint
	for rows.Next() {
		var entry cards.CardPrint
		err = rows.Scan(
			&entry.Name,
			&entry.ID.CardID,
			&entry.ID.FaceID,
			&entry.Number,
			&entry.Code,
			&entry.Amount,
		)
		if err != nil {
			return cards.CardPrints{}, fmt.Errorf("failed to execute card print scan after select %w", err)
		}
		result = append(result, entry)
	}
	if rows.Err() != nil {
		return cards.CardPrints{}, fmt.Errorf("failed to read next row %w", rows.Err())
	}

	return cards.NewCardPrints(result, page), nil
}

func (r *PostgresCardRepository) Exist(ctx context.Context, id cards.ID) (bool, error) {
	if id.CardID < 0 {
		return false, fmt.Errorf("exist failed for card ID %v, %w", id.CardID, cards.ErrInvalidID)
	}
	args := pgx.NamedArgs{
		"id": id.CardID,
	}
	query := `
SELECT
  c.id
FROM
  card AS c
WHERE
  c.id = @id`
	row := r.db.Conn.QueryRow(ctx, query, args)

	var cardID int
	if err := row.Scan(&cardID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("exist failed during row scan %w", err)
	}

	return true, nil
}

func (r *PostgresCardRepository) Collect(ctx context.Context, item cards.Collectable, c cards.Collector) error {
	args := pgx.NamedArgs{
		"cardID": item.ID.CardID,
		"amount": item.Amount,
		"userID": c.ID,
	}
	query := `
INSERT INTO
  card_collection (card_id, amount, user_id)
VALUES 
  (@cardID, @amount, @userID)
ON CONFLICT
  (card_id, user_id)
DO UPDATE SET
  amount = excluded.amount`
	if _, err := r.db.Conn.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("collect failed due to exec error %w", err)
	}

	return nil
}

func (r *PostgresCardRepository) Remove(ctx context.Context, item cards.Collectable, c cards.Collector) error {
	args := pgx.NamedArgs{
		"cardID": item.ID.CardID,
		"userID": c.ID,
	}
	query := `
DELETE FROM
  card_collection
WHERE
  card_id = @cardID
AND
  user_id = @userID`
	if _, err := r.db.Conn.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("remove failed due to exec error %w", err)
	}

	return nil
}
