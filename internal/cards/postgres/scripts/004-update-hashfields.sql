DROP INDEX IF EXISTS idx_card_image_hashes;

ALTER TABLE card_image
    ADD COLUMN phash_r BIT(256),
    ADD COLUMN phash_g BIT(256),
    ADD COLUMN phash_b BIT(256),
    ADD COLUMN dhash   BIT(64);

ALTER TABLE card_image
    DROP COLUMN phash1,
    DROP COLUMN phash2,
    DROP COLUMN phash3,
    DROP COLUMN phash4;

CREATE INDEX idx_card_image_hashes ON card_image (phash_r, phash_g, phash_b, dhash);
