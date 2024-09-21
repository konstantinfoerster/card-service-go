package postgres

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/jackc/pgx/v5"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/pkg/errors"
)

type postgresCardRepository struct {
	db  *DBConnection
	cfg config.Images
}

func NewCardRepository(connection *DBConnection, cfg config.Images) cards.CardRepository {
	return &postgresCardRepository{
		db:  connection,
		cfg: cfg,
	}
}

func (r postgresCardRepository) Find(ctx context.Context, f cards.Filter, page cards.Page) (cards.Cards, error) {
	args := pgx.NamedArgs{
		"limit":   page.Size(),
		"offset":  page.Offset(),
		"baseURL": r.cfg.Host,
		"lang":    cards.DefaultLang,
	}

	tplParams := make(map[string]any)
	if f.Name != "" {
		args["name"] = f.Name
		tplParams["name"] = f.Name
	}

	if f.Collector != nil {
		args["user"] = f.Collector.ID
		tplParams["user"] = f.Collector.ID
		tplParams["onlyCollected"] = f.OnlyCollected
	}

	qt, err := template.New("selectcard").
		Parse(`
		WITH
          cte
        AS (
            SELECT
                 DISTINCT ON (face.name)
                 row_number() over (partition by face.card_id) as rn,
                 face.card_id, face.name, set.code, set.name,
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
                card.id = image.card_id
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
            {{if .name}}
                    (face.name ILIKE '%' || @name || '%')
                AND
            {{end}}
                (face.id = image.face_id OR image.face_id IS NULL) 
            AND 
                (image.lang_lang = @lang OR image.lang_lang IS NULL)
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
		return cards.Empty(page), fmt.Errorf("failed to parse card select template %w", err)
	}
	var query bytes.Buffer
	if err = qt.Execute(&query, tplParams); err != nil {
		return cards.Empty(page), fmt.Errorf("failed to execute query template %w", err)
	}

	rows, err := r.db.Conn.Query(ctx, query.String(), args)
	if err != nil {
		return cards.Empty(page), fmt.Errorf("failed to execute paged card face select %w", err)
	}
	defer rows.Close()

	var result []cards.Card
	for rows.Next() {
		var entry dbCard
		var rn int
		err = rows.Scan(
			&rn,
			&entry.CardID,
			&entry.Name,
			&entry.SetCode,
			&entry.SetName,
			&entry.ImageURL,
			&entry.Amount,
		)
		if err != nil {
			return cards.Empty(page), fmt.Errorf("failed to execute card scan after select %w", err)
		}
		result = append(result, toCard(entry))
	}
	if rows.Err() != nil {
		return cards.Empty(page), fmt.Errorf("failed to read next row %w", rows.Err())
	}

	return cards.NewCards(result, page), nil
}

func (r postgresCardRepository) Exist(ctx context.Context, id int) (bool, error) {
	args := pgx.NamedArgs{
		"id": id,
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

		return false, fmt.Errorf("failed to execute card scan after select %w", err)
	}

	return true, nil
}

func (r postgresCardRepository) Collect(ctx context.Context, item cards.Collectable, c cards.Collector) error {
	query := `
		INSERT INTO
			card_collection (card_id, amount, user_id)
		VALUES 
			($1, $2, $3)
		ON CONFLICT
			(card_id, user_id)
		DO UPDATE SET
			amount = excluded.amount`
	if _, err := r.db.Conn.Exec(ctx, query, item.ID, item.Amount, c.ID); err != nil {
		return err
	}

	return nil
}

func (r postgresCardRepository) Remove(ctx context.Context, item cards.Collectable, c cards.Collector) error {
	query := `
		DELETE FROM
			card_collection
		WHERE
			card_id = $1
		AND
			user_id = $2`
	if _, err := r.db.Conn.Exec(ctx, query, item.ID, c.ID); err != nil {
		return err
	}

	return nil
}
