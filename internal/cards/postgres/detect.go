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

	sumExprs := make([]string, 0, len(hashes))
	condExprs := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		queryArgs = append(queryArgs, hash.PHashRBase2())
		rIdx := len(queryArgs)
		queryArgs = append(queryArgs, hash.PHashGBase2())
		gIdx := len(queryArgs)
		queryArgs = append(queryArgs, hash.PHashBBase2())
		bIdx := len(queryArgs)
		queryArgs = append(queryArgs, hash.DHashBase2())
		dIdx := len(queryArgs)

		rExpr := fmt.Sprintf("BIT_COUNT(image.phash_r # $%d)", rIdx)
		gExpr := fmt.Sprintf("BIT_COUNT(image.phash_g # $%d)", gIdx)
		bExpr := fmt.Sprintf("BIT_COUNT(image.phash_b # $%d)", bIdx)
		dExpr := fmt.Sprintf("BIT_COUNT(image.dhash # $%d)", dIdx)

		sumExprs = append(sumExprs, fmt.Sprintf("(%s+%s+%s+%s)", rExpr, gExpr, bExpr, dExpr))
		// a candidate hash only matches if every field individually clears its own threshold
		condExprs = append(condExprs, fmt.Sprintf(
			"(%s < %d AND %s < %d AND %s < %d AND %s < %d)",
			rExpr, cards.PHashThreshold, gExpr, cards.PHashThreshold, bExpr, cards.PHashThreshold, dExpr, cards.DHashThreshold,
		))
	}

	scoreExpr := "LEAST(" + strings.Join(sumExprs, ",") + ")"
	whereExpr := strings.Join(condExprs, " OR ")

	query := fmt.Sprintf(`
SELECT
  image.card_id, image.face_id, %s
FROM
  card_image as image
WHERE
  %s
GROUP BY
  image.card_id, image.face_id, image.phash_r, image.phash_g, image.phash_b, image.dhash
ORDER BY
  %s, image.card_id, image.face_id
LIMIT $1`, scoreExpr, whereExpr, scoreExpr)
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
