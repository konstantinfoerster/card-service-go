package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type PostgresDetectRepository struct {
	db  *DBConnection
	cfg Images
}

func NewDetectRepository(connection *DBConnection, cfg Images) *PostgresDetectRepository {
	return &PostgresDetectRepository{
		db:  connection,
		cfg: cfg,
	}
}

func (r *PostgresDetectRepository) Top5MatchesByHash(ctx context.Context, hashes ...cards.Hash) (cards.Scores, error) {
	defer cards.TimeTracker(time.Now(), "Top5MatchesByHash")
	if len(hashes) == 0 {
		return cards.Scores{}, nil
	}

	limit := 5
	queryArgs := []any{limit}
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
			queryArgs = append(queryArgs, v)
			sb.WriteString(fmt.Sprintf("BIT_COUNT(image.phash%d # $%d)", x+1, len(queryArgs)))
		}
		sb.WriteString(")")
	}
	sb.WriteString(")")

	query := fmt.Sprintf(`
SELECT
  image.card_id, image.face_id, %s
FROM
  card_image as image
WHERE
  CAST(%s as int) < 60
GROUP BY
  image.card_id, image.face_id, image.phash1, image.phash2, image.phash3, image.phash4
ORDER BY
  image.face_id, %s
LIMIT $1`, sb.String(), sb.String(), sb.String())
	rows, err := r.db.Conn.Query(ctx, query, queryArgs...)
	if err != nil {
		return cards.Scores{}, fmt.Errorf("failed to execute top 5 phash select %w", err)
	}
	defer rows.Close()

	var result cards.Scores
	for rows.Next() {
		entry := cards.Score{}
		if err = rows.Scan(&entry.ID.CardID, &entry.ID.FaceID, &entry.Score); err != nil {
			return cards.Scores{}, fmt.Errorf("failed to execute top 5 phash result scan %w", err)
		}
		result = append(result, entry)
	}
	if rows.Err() != nil {
		return cards.Scores{}, fmt.Errorf("failed to read next top 5 hash row %w", rows.Err())
	}

	return result, err
}
