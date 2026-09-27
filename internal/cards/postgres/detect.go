package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detection"
)

// PHashThreshold max Hamming distance for a 256-bit pHash to count as a match.
const PHashThreshold = 60

// DHashThreshold max Hamming distance for a dhash to count as a match.
const DHashThreshold = 28

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

func (r *PostgresDetectRepository) Top5MatchesByHash(
	ctx context.Context, hashes ...detection.Hash) (cards.Scores, error) {
	defer cards.TimeTracker(time.Now(), "Top5MatchesByHash")
	if len(hashes) == 0 {
		return cards.Scores{}, nil
	}

	limit := 5
	queryArgs := []any{limit}

	sumExprs := make([]string, 0, len(hashes))
	condExprs := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		queryArgs = append(queryArgs, asBase2(hash.PHashR...))
		rIdx := len(queryArgs)
		queryArgs = append(queryArgs, asBase2(hash.PHashG...))
		gIdx := len(queryArgs)
		queryArgs = append(queryArgs, asBase2(hash.PHashB...))
		bIdx := len(queryArgs)
		queryArgs = append(queryArgs, asBase2(hash.DHash))
		dIdx := len(queryArgs)

		rExpr := fmt.Sprintf("BIT_COUNT(image.phash_r # $%d)", rIdx)
		gExpr := fmt.Sprintf("BIT_COUNT(image.phash_g # $%d)", gIdx)
		bExpr := fmt.Sprintf("BIT_COUNT(image.phash_b # $%d)", bIdx)
		dExpr := fmt.Sprintf("BIT_COUNT(image.dhash # $%d)", dIdx)

		sumExprs = append(sumExprs, fmt.Sprintf("(%s+%s+%s+%s)", rExpr, gExpr, bExpr, dExpr))
		// a candidate hash only matches if every field individually clears its own threshold
		condExprs = append(condExprs, fmt.Sprintf(
			"(%s < %d AND %s < %d AND %s < %d AND %s < %d)",
			rExpr, PHashThreshold, gExpr, PHashThreshold, bExpr, PHashThreshold, dExpr, DHashThreshold,
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

// asBase2 concatenates the given 64-bit words into a single base-2 string.
func asBase2(words ...uint64) string {
	var sb strings.Builder
	for _, v := range words {
		fmt.Fprintf(&sb, "%064b", v)
	}

	return sb.String()
}
