package postgres

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detection"
)

type PostgresDetectRepository struct {
	db *DBConnection
}

func NewDetectRepository(connection *DBConnection) *PostgresDetectRepository {
	return &PostgresDetectRepository{
		db: connection,
	}
}

// AllHashes returns the hashes of all card images that have them.
func (r *PostgresDetectRepository) AllHashes(ctx context.Context) ([]detection.ImageHash, error) {
	defer cards.TimeTracker(time.Now(), "AllHashes")

	query := `
SELECT
  image.card_id, image.face_id, card.name, image.phash_r, image.phash_g, image.phash_b, image.dhash
FROM
  card_image AS image
INNER JOIN
  card ON card.id = image.card_id
WHERE
  image.phash_r IS NOT NULL
  AND image.phash_g IS NOT NULL
  AND image.phash_b IS NOT NULL
  AND image.dhash IS NOT NULL`
	rows, err := r.db.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute image hash select %w", err)
	}
	defer rows.Close()

	var result []detection.ImageHash
	for rows.Next() {
		var img detection.ImageHash
		var red, green, blue, dhash pgtype.Bits
		if err = rows.Scan(&img.ID.CardID, &img.ID.FaceID, &img.Name, &red, &green, &blue, &dhash); err != nil {
			return nil, fmt.Errorf("failed to execute image hash result scan %w", err)
		}

		// the columns have a fixed size, bit(256) for a pHash and bit(64) for the dHash
		img.Hash = detection.Hash{
			PHashR: asWords(red),
			PHashG: asWords(green),
			PHashB: asWords(blue),
			DHash:  asWords(dhash)[0],
		}
		result = append(result, img)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read next image hash row %w", rows.Err())
	}

	return result, nil
}

// asWords splits the bit string into 64-bit words.
func asWords(b pgtype.Bits) []uint64 {
	words := make([]uint64, len(b.Bytes)/8)
	for i := range words {
		words[i] = binary.BigEndian.Uint64(b.Bytes[i*8:])
	}

	return words
}
