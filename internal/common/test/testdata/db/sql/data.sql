-- Card 1
INSERT INTO
    card(name, number, rarity, border, layout, card_set_code)
    VALUES ('Dummy Card 1', '1', 'COMMON', 'WHITE', 'NORMAL', 'M10');
INSERT INTO
    card_face(card_id, name, converted_mana_cost)
VALUES (1, 'Dummy Card 1', 0);
INSERT INTO
    card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (1, 1, 'images/dummyCard1.png', 'eng', 'jpeg');

-- Card 2
INSERT INTO
    card(name, number, rarity, border, layout, card_set_code)
VALUES ('Dummy Card 2', '2', 'COMMON', 'WHITE', 'NORMAL', 'M10');
INSERT INTO
    card_face(card_id, name, converted_mana_cost)
VALUES (2, 'Dummy Card 2', 0);
INSERT INTO
    card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (2, 2, 'images/dummyCard2.png', 'eng', 'jpeg');

-- Card 3
INSERT INTO
    card(name, number, rarity, border, layout, card_set_code)
VALUES ('Dummy Card 3', '3', 'COMMON', 'WHITE', 'NORMAL', 'M10');
INSERT INTO
    card_face(card_id, name, converted_mana_cost)
VALUES (3, 'Dummy Card 3', 0);
INSERT INTO
    card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (3, 3, 'images/dummyCard3.png', 'eng', 'jpeg');

-- Card 4
INSERT INTO
    card(name, number, rarity, border, layout, card_set_code)
VALUES ('Dummy Card 4', '4', 'COMMON', 'WHITE', 'NORMAL', 'M10');
INSERT INTO
    card_face(card_id, name, converted_mana_cost)
VALUES (4, 'Dummy Card 4', 0);
INSERT INTO
    card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (4, 4, 'images/dummyCard4.png', 'eng', 'jpeg');

-- Card no image
INSERT INTO
    card(name, number, rarity, border, layout, card_set_code)
VALUES ('No Image Card 1', '1', 'COMMON', 'WHITE', 'NORMAL', 'M11');
INSERT INTO
    card_face(card_id, name, converted_mana_cost)
VALUES (5, 'No Image Card 1', 0);

-- Card no image face linked
INSERT INTO
    card(name, number, rarity, border, layout, card_set_code)
VALUES ('No Image Card 2', '2', 'COMMON', 'WHITE', 'NORMAL', 'M11');
INSERT INTO
    card_face(card_id, name, converted_mana_cost)
VALUES (6, 'No Image Card 2', 0);
INSERT INTO
    card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (null, 6, 'images/noFace.png', 'eng', 'jpeg');

-- Card language fra
INSERT INTO
    card(name, number, rarity, border, layout, card_set_code)
VALUES ('French', '1', 'COMMON', 'WHITE', 'NORMAL', 'M12');
INSERT INTO
    card_face(card_id, name, converted_mana_cost)
VALUES (7, 'French', 0);
INSERT INTO
    card_image(face_id, card_id, image_path, lang_lang, mime_type)
VALUES (7, 7, 'images/French.png', 'fra', 'jpeg');