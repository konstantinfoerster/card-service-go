package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/image"
)

type DetectRepository interface {
	Top5MatchesByHash(ctx context.Context, hashes ...image.Hash) (cards.Matches, error)
	Top5MatchesByCollectorAndHash(ctx context.Context, c cards.Collector, hashes ...image.Hash) (cards.Matches, error)
}

type postgresDetectRepository struct {
	db  *DBConnection
	cfg config.Images
}

func NewDetectRepository(connection *DBConnection, cfg config.Images) DetectRepository {
	return &postgresDetectRepository{
		db:  connection,
		cfg: cfg,
	}
}

func (r *postgresDetectRepository) Top5MatchesByHash(ctx context.Context,
	hashes ...image.Hash) (cards.Matches, error) {
	defer cards.TimeTracker(time.Now(), "Top5MatchesByHash")
	if len(hashes) == 0 {
		return cards.Matches{}, nil
	}

	limit := 5
	params := []any{limit, r.cfg.Host, cards.DefaultLang}
	var sb strings.Builder
	sb.WriteString("LEAST(")
	for i, hash := range hashes {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("(")
		for x, v := range hash.AsBase2() {
			if x > 0 {
				sb.WriteString("+")
			}
			params = append(params, v)
			sb.WriteString(fmt.Sprintf("BIT_COUNT(image.phash%d # $%d)", x+1, len(params)))
		}
		sb.WriteString(")")
	}
	sb.WriteString(")")

	query := fmt.Sprintf(`
		SELECT
			 face.card_id, face.name,
		     NULLIF(CONCAT($2::text, max(image.image_path)), $2::text),
             %s as score
		FROM
			card_face AS face
	    JOIN
			card_image as image
		ON
			face.card_id = image.card_id
		AND
			(face.id = image.face_id OR image.face_id IS NULL) 
		WHERE
            (image.lang_lang = $3 OR image.lang_lang IS NULL) AND
            CAST(%s as int) < 60
		GROUP BY
			face.card_id, face.name, image.phash1, image.phash2, image.phash3, image.phash4
		ORDER BY
			%s, face.name
		LIMIT $1`, sb.String(), sb.String(), sb.String())
	rows, err := r.db.Conn.Query(ctx, query, params...)
	if err != nil {
		return cards.Matches{}, fmt.Errorf("failed to execute top 5 phash select %w", err)
	}
	defer rows.Close()

	var result cards.Matches
	for rows.Next() {
		var entry matchDBCard
		if err = rows.Scan(&entry.CardID, &entry.Name, &entry.Image, &entry.Score); err != nil {
			return cards.Matches{}, fmt.Errorf("failed to execute card scan after select %w", err)
		}
		result = append(result, toCardMatch(&entry))
	}
	if rows.Err() != nil {
		return cards.Matches{}, fmt.Errorf("failed to read next row %w", rows.Err())
	}

	return result, err
}

func (r *postgresDetectRepository) Top5MatchesByCollectorAndHash(ctx context.Context,
	c cards.Collector, hashes ...image.Hash) (cards.Matches, error) {
	defer cards.TimeTracker(time.Now(), "Top5MatchesByHashAndCollector")

	if len(hashes) == 0 {
		return cards.Matches{}, nil
	}

	limit := 5
	params := []any{limit, r.cfg.Host, cards.DefaultLang, c.ID}
	var sb strings.Builder
	sb.WriteString("LEAST(")
	for i, hash := range hashes {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("(")
		for x, v := range hash.AsBase2() {
			if x > 0 {
				sb.WriteString("+")
			}
			params = append(params, v)
			sb.WriteString(fmt.Sprintf("BIT_COUNT(image.phash%d # $%d)", x+1, len(params)))
		}
		sb.WriteString(")")
	}
	sb.WriteString(")")

	query := fmt.Sprintf(`
		SELECT
			face.card_id, face.name,
		    NULLIF(CONCAT($2::text, max(image.image_path)), $2::text),
            coalesce(min(card_collection.amount), 0),
            %s as score
		FROM
			card_face AS face
		LEFT JOIN
			card_collection
		ON
			face.card_id = card_collection.card_id
		AND
			card_collection.user_id = $4
		LEFT JOIN
			card_image as image
		ON
			face.card_id = image.card_id
		AND
			(face.id = image.face_id OR image.face_id IS NULL) 
		WHERE
            (image.lang_lang = $3 OR image.lang_lang IS NULL) AND
            image.phash1 IS NOT NULL AND
            CAST(%s as int) < 60
		GROUP BY
			face.card_id, face.name, image.phash1, image.phash2, image.phash3, image.phash4
		ORDER BY
			%s, face.name
		LIMIT $1`, sb.String(), sb.String(), sb.String())
	rows, err := r.db.Conn.Query(ctx, query, params...)
	if err != nil {
		return cards.Matches{}, fmt.Errorf("failed to execute top 5 phash with collector select %w", err)
	}
	defer rows.Close()

	var result cards.Matches
	for rows.Next() {
		var entry matchDBCard
		if err = rows.Scan(&entry.CardID, &entry.Name, &entry.Image, &entry.Amount, &entry.Score); err != nil {
			return cards.Matches{}, fmt.Errorf("failed to execute card scan after select %w", err)
		}
		result = append(result, toCardMatch(&entry))
	}
	if rows.Err() != nil {
		return cards.Matches{}, fmt.Errorf("failed to read next row %w", rows.Err())
	}

	return result, err
}

type matchDBCard struct {
	dbCard
	Score int
}

func toCardMatch(dbCard *matchDBCard) cards.Match {
	return cards.Match{
		Card: cards.Card{
			ID:     dbCard.CardID,
			Name:   dbCard.Name,
			Image:  dbCard.Image.String,
			Amount: dbCard.Amount,
		},
		Score: dbCard.Score,
	}
}
