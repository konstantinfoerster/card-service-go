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

-- Card 4
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Dummy Card 4', '4', 'COMMON', 'WHITE', 'NORMAL', 'M10');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (4, 'Dummy Card 4', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (4, 4, 'images/dummyCard4.png', 'eng', 'png');

-- Card has no image
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('No Image Card 1', '1', 'COMMON', 'WHITE', 'NORMAL', 'M11');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (5, 'No Image Card 1', 0);
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (5, 'myUser', 5);

-- Card image has no face id
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('No Image Card 2', '2', 'COMMON', 'WHITE', 'NORMAL', 'M11');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (6, 'No Image Card 2', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (null, 6, 'images/noFace.png', 'eng', 'png');
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (6, 'myUser', 1);

-- Card language fra
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('French', '1', 'COMMON', 'WHITE', 'NORMAL', 'M12');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (7, 'French', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (7, 7, 'images/French.png', 'fra', 'png');

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

-- Card is not collected
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Uncollected Card 1', '1', 'COMMON', 'WHITE', 'NORMAL', 'M14');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (9, 'Uncollected Card 1', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (10, 9, 'images/uncollectedCard1.png', 'eng', 'png');

-- Card will be removed from collection
INSERT INTO card(name, number, rarity, border, layout, card_set_code)
VALUES ('Remove Collected Card 1', '2', 'COMMON', 'WHITE', 'NORMAL', 'M14');
INSERT INTO card_face(card_id, name, converted_mana_cost)
VALUES (10, 'Remove Collected Card 1', 0);
INSERT INTO card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (11, 10, 'images/removeCollectedCard1.png', 'eng', 'png');
INSERT INTO card_collection(card_id, user_id, amount)
VALUES (10, 'myUser', 2);
