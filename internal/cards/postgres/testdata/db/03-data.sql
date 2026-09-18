-- card sets
INSERT INTO card_set(code, name, type, total_count)
VALUES ('M10', 'Magic 2010', 'CORE', 100);

INSERT INTO card_set(code, name, type, total_count)
VALUES ('M11', 'Magic 2011', 'CORE', 100);

INSERT INTO card_set(code, name, type, total_count)
VALUES ('M12', 'Magic 2012', 'CORE', 100);

INSERT INTO card_set(code, name, type, total_count)
VALUES ('M13', 'Magic 2013', 'CORE', 100);

INSERT INTO card_set(code, name, type, total_count)
VALUES ('M14', 'Magic 2014', 'CORE', 100);

INSERT INTO card_set(code, name, type, total_count)
VALUES ('M15', 'Magic 2015', 'CORE', 100);

INSERT INTO card_set(code, name, type, total_count)
VALUES ('M16', 'Magic 2016', 'CORE', 100);

-- Card 1
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Dummy Card 1', '1', 'COMMON', 'WHITE', 'NORMAL', 'M10');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (1, 'Dummy Card 1', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (1, 1, 'images/dummyCard1.png', 'eng', 'png');
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (1, 'myUser', 3);

-- Card 2
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Dummy Card 2', '2', 'COMMON', 'WHITE', 'NORMAL', 'M10');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (2, 'Dummy Card 2', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (2, 2, 'images/dummyCard2.png', 'eng', 'png');
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (2, 'myUser', 1);

-- Card 3
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Dummy Card 3', '3', 'COMMON', 'WHITE', 'NORMAL', 'M10');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (3, 'Dummy Card 3', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (3, 3, 'images/dummyCard3.png', 'eng', 'png');
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (3, 'otherUser', 2);

-- Card 4
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Dummy Card 4', '4', 'COMMON', 'WHITE', 'NORMAL', 'M10');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (4, 'Dummy Card 4', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (4, 4, 'images/dummyCard4.png', 'eng', 'png');

-- Card 5 has no image
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('No Image Card 1', '1', 'COMMON', 'WHITE', 'NORMAL', 'M11');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (5, 'No Image Card 1', 0);
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (5, 'myUser', 5);

-- Card 6 image has no face id
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('No Image Card 2', '2', 'COMMON', 'WHITE', 'NORMAL', 'M11');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (6, 'No Image Card 2', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (null, 6, 'images/noFaceID.png', 'eng', 'png');
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (6, 'myUser', 1);

-- Card 7 language fra
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('French', '1', 'COMMON', 'WHITE', 'NORMAL', 'M12');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (7, 'French', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (7, 7, 'images/French.png', 'fra', 'png');

--- Card 8, face 8, face 9 with multiple faces
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Double Face', '1', 'COMMON', 'WHITE', 'NORMAL', 'M13');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (8, 'Front Face doubleFace', 0);
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (8, 'Back Face doubleFace', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (8, 8, 'images/FrontFace.png', 'eng', 'png');
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (9, 8, 'images/BackFace.png', 'eng', 'png');
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (8, 'myUser', 2);
-- Card 9, face 10 is not collected
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Uncollected Card 1', '1', 'COMMON', 'WHITE', 'NORMAL', 'M14');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (9, 'Uncollected Card 1', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (10, 9, 'images/uncollectedCard1.png', 'eng', 'png');

-- Card 10, face 11 will be removed from collection
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Remove Collected Card 1', '2', 'COMMON', 'WHITE', 'NORMAL', 'M14');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (10, 'Remove Collected Card 1', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (11, 10, 'images/removeCollectedCard1.png', 'eng', 'png');
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (10, 'myUser', 2);

-- Card 11 with hash
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Card 11 with hash', '1', 'COMMON', 'WHITE', 'NORMAL', 'M15');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (11, 'Card 11 with hash', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type,
phash_r, phash_g, phash_b, dhash)
VALUES (12, 11, 'images/card11Hash.png', 'eng', 'png',
'1000000101010001001101101000000100111110110011101001110001010100011110101000010111000010101001000110001110101010111000010000111001101111000011100000111101111010001110111111101000011000111001110011110010100011001001001110100100101111111010010001111011101001',
'1000000101010001001101101000000100111110110011101001110001010100011110101000010111000010101001000110001110101010111000010000111001101111000011100000111101111010001110111111101000011000111001110011110010100011001001001110100100101111111010010001111011101001',
'1000000101010001001101101000000100111110110011101001110001010100011110101000010111000010101001000110001110101010111000010000111001101111000011100000111101111010001110111111101000011000111001110011110010100011001001001110100100101111111010010001111011101001',
'0010010110100101101001011010010110100101101001011010010110100101'
);
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (11, 'myUser', 3);

-- Card 12 with hash
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Card 12 with hash', '2', 'COMMON', 'WHITE', 'NORMAL', 'M15');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (12, 'Card 12 with hash', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type,
phash_r, phash_g, phash_b, dhash)
VALUES (13, 12, 'images/card12hash.png', 'eng', 'png',
'1000000101010101001101101000000100111110110011101001110001010100011110101000010111000010101001000110001110101010111000010000111001101111001011100000101101111010001110111111101000011000111001010011110010100011001001011110100100101111111010010001111011100001',
'1000000101010101001101101000000100111110110011101001110001010100011110101000010111000010101001000110001110101010111000010000111001101111001011100000101101111010001110111111101000011000111001010011110010100011001001011110100100101111111010010001111011100001',
'1000000101010101001101101000000100111110110011101001110001010100011110101000010111000010101001000110001110101010111000010000111001101111001011100000101101111010001110111111101000011000111001010011110010100011001001011110100100101111111010010001111011100001',
'0110010110100101101001011010010110100101101001011010010110100101'
);
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (12, 'myUser', 1);

-- Card 13 with hash
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Card 13 with hash', '3', 'COMMON', 'WHITE', 'NORMAL', 'M15');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (13, 'Card 13 with hash', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type,
phash_r, phash_g, phash_b, dhash)
VALUES (14, 13, 'images/card13hash.png', 'eng', 'png',
'1000010101010001001101101000000100111110110010101001110001010100011110101000010111000010101001000110001110101110110000010010111001101111001011100000111101111010001110111111101000011000111001110011110010000011001001001110100100101111111010010001101011101001',
'1000010101010001001101101000000100111110110010101001110001010100011110101000010111000010101001000110001110101110110000010010111001101111001011100000111101111010001110111111101000011000111001110011110010000011001001001110100100101111111010010001101011101001',
'1000010101010001001101101000000100111110110010101001110001010100011110101000010111000010101001000110001110101110110000010010111001101111001011100000111101111010001110111111101000011000111001110011110010000011001001001110100100101111111010010001101011101001',
'0100010110100101101001011010010110100101101001011010010110100101'
);

-- Card 14 with hash, that card is too far away in both phash and dhash to match
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Card 14 with hash', '4', 'COMMON', 'WHITE', 'NORMAL', 'M15');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (14, 'Card 14 with hash', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type,
phash_r, phash_g, phash_b, dhash)
VALUES (15, 14, 'images/card14hash.png', 'eng', 'png',
'1001010100010001001111100000000101111010111010101001000001111010011110111000001011100001101100100110001110101110111000110100110001101110000011101100011000111000000011101011101100111100101100111001110011100011100111001010001100011011101011100001101101001100',
'1001010100010001001111100000000101111010111010101001000001111010011110111000001011100001101100100110001110101110111000110100110001101110000011101100011000111000000011101011101100111100101100111001110011100011100111001010001100011011101011100001101101001100',
'1001010100010001001111100000000101111010111010101001000001111010011110111000001011100001101100100110001110101110111000110100110001101110000011101100011000111000000011101011101100111100101100111001110011100011100111001010001100011011101011100001101101001100',
'0101101001011010010101011010010110100101101001011010010110100101'
);
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (14, 'myUser', 1);


-- Card 15 with same name as card 1 but different set
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Dummy Card 1', '10', 'COMMON', 'WHITE', 'NORMAL', 'M16');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (15, 'Dummy Card 1', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (16, 15, 'images/dummyCard15.png', 'eng', 'png');
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (15, 'myUser', 1);

-- Card 16 with same name as card 1 but different set
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Baa Card', '20', 'COMMON', 'WHITE', 'NORMAL', 'M16');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (16, 'Baa Card', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (17, 16, 'images/dummyCard16.png', 'eng', 'png');

-- Card 17 with same name as card 1 but different set
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Aa Card', '30', 'COMMON', 'WHITE', 'NORMAL', 'M16');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (17, 'Aa Card', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (18, 17, 'images/dummyCard17.png', 'eng', 'png');

-- Collected card that does not exists
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (99999, 'myOtherUser', 1);

-- Card 18 first print
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Print Card', '31', 'COMMON', 'WHITE', 'NORMAL', 'M16');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (18, 'Print Card', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (19, 18, 'images/dummyCard18.png', 'eng', 'png');
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (18, 'myUser', 3);

-- Card 19 second print
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Print Card', '32', 'COMMON', 'WHITE', 'NORMAL', 'M15');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (19, 'Print Card', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (20, 19, 'images/dummyCard19.png', 'eng', 'png');


-- Card 20 third print
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Print Card', '33', 'COMMON', 'WHITE', 'NORMAL', 'M14');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (20, 'Print Card', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (21, 20, 'images/dummyCard20.png', 'eng', 'png');

