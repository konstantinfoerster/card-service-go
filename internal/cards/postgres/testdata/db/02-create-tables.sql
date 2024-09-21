CREATE TABLE lang
(
    lang CHAR(3) PRIMARY KEY NOT NULL CHECK (lang = lower(lang) AND lang <> '') -- <> == not equal
);

INSERT INTO lang
VALUES ('deu');
INSERT INTO lang
VALUES ('eng');
INSERT INTO lang
VALUES ('fra');

-- Border --
CREATE TYPE border AS ENUM (
    'WHITE',
    'BLACK',
    'SILVER',
    'GOLD',
    'BORDERLESS'
    );

-- Card set type --
CREATE TYPE card_set_type AS ENUM (
    'CORE',
    'EXPANSION',
    'REPRINT',
    'BOX',
    'UN',
    'FROM_THE_VAULT',
    'PREMIUM_DECK',
    'DUEL_DECK',
    'STARTER',
    'COMMANDER',
    'PLANECHASE',
    'ARCHENEMY',
    'PROMO',
    'VANGUARD',
    'MASTERS',
    'MEMORABILIA',
    'DRAFT_INNOVATION',
    'FUNNY',
    'MASTERPIECE',
    'TOKEN',
    'TREASURE_CHEST',
    'SPELLBOOK',
    'ARSENAL',
    'ALCHEMY'
    );

-- Layout --
CREATE TYPE layout AS ENUM (
    'NORMAL',
    'SPLIT',
    'FLIP',
    'TOKEN',
    'PLANE',
    'SCHEMA',
    'PHENOMENON',
    'LEVELER',
    'VANGUARD',
    'MELD',
    'AFTERMATH',
    'SAGA',
    'TRANSFORM',
    'ADVENTURE',
    'MODAL_DFC',
    'SCHEME',
    'PLANAR',
    'HOST',
    'AUGMENT',
    'CLASS',
    'REVERSIBLE_CARD'
    );

-- Rarity --
CREATE TYPE rarity AS ENUM (
    'COMMON',
    'UNCOMMON',
    'RARE',
    'MYTHIC',
    'SPECIAL',
    'BASIC_LAND',
    'BONUS'
    );

-- Sub Type --
CREATE TABLE sub_type
(
    id   INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL CHECK ( name <> '' ),
    UNIQUE (name)
);

CREATE TABLE sub_type_translation
(
    id          INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        VARCHAR(100) NOT NULL CHECK ( name <> '' ),
    lang_lang   CHAR(3) REFERENCES lang (lang),
    sub_type_id INTEGER REFERENCES sub_type (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, sub_type_id)
);

-- Super Type --
CREATE TABLE super_type
(
    id   INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL CHECK ( name <> '' ),
    UNIQUE (name)
);

CREATE TABLE super_type_translation
(
    id            INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          VARCHAR(100) NOT NULL CHECK ( name <> '' ),
    lang_lang     CHAR(3) REFERENCES lang (lang),
    super_type_id INTEGER REFERENCES super_type (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, super_type_id)
);

-- Card Type --
CREATE TABLE card_type
(
    id   INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL CHECK ( name <> '' ),
    UNIQUE (name)
);

CREATE TABLE card_type_translation
(
    id           INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name         VARCHAR(100) NOT NULL CHECK ( name <> '' ),
    lang_lang    CHAR(3) REFERENCES lang (lang),
    card_type_id INTEGER REFERENCES card_type (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, card_type_id)
);

-- Card Block --
CREATE TABLE card_block
(
    id    INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    block VARCHAR(255) NOT NULL UNIQUE CHECK ( block <> '' )
);

CREATE TABLE card_block_translation
(
    id            INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    block         VARCHAR(255) NOT NULL CHECK ( block <> '' ),
    lang_lang     CHAR(3) REFERENCES lang (lang),
    card_block_id INTEGER REFERENCES card_block (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, card_block_id)
);

-- Card Set --
CREATE TABLE card_set
(
    code          VARCHAR(10) PRIMARY KEY NOT NULL CHECK ( code <> '' AND code = upper(code)),
    name          VARCHAR(255)            NOT NULL CHECK ( name <> '' ),
    type          card_set_type           NOT NULL, -- Enum
    released      DATE,                             -- TODO check if not null is possible
    total_count   INTEGER                 NOT NULL CHECK ( total_count >= 0 ),
    card_block_id INTEGER REFERENCES card_block (id),
    UNIQUE (code, card_block_id)
);

CREATE TABLE card_set_translation
(
    id            INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          VARCHAR(255) NOT NULL CHECK ( name <> '' ),
    lang_lang     CHAR(3) REFERENCES lang (lang),
    card_set_code VARCHAR(10) REFERENCES card_set (code) ON DELETE CASCADE,
    UNIQUE (lang_lang, card_set_code)
);

-- Card --
CREATE TABLE card
(
    id            INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          VARCHAR(255) NOT NULL CHECK ( name <> '' ),
    number        VARCHAR(255) NOT NULL CHECK ( number <> '' ),
    rarity        rarity       NOT NULL, -- Enum
    border        border       NOT NULL, -- Enum
    layout        layout       NOT NULL, -- Enum
    card_set_code VARCHAR(10)  NOT NULL CHECK ( card_set_code <> '' AND card_set_code = upper(card_set_code) ),
    unique (card_set_code, number)
);


CREATE TABLE card_face
(
    id                  INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name                VARCHAR(255)   NOT NULL CHECK ( name <> '' ),
    text                VARCHAR(800),
    flavor_text         VARCHAR(500),
    type_line           VARCHAR(255),
    converted_mana_cost NUMERIC(10, 2) NOT NULL CHECK ( converted_mana_cost >= 0 ),
    colors              VARCHAR(100),                                                -- List of Strings with ',' as separator
    artist              VARCHAR(100),
    hand_modifier       VARCHAR(10),                                                 -- only Vanguard cards
    life_modifier       VARCHAR(10),                                                 -- only Vanguard cards
    loyalty             VARCHAR(10),                                                 -- only planeswalker
    mana_cost           VARCHAR(255),
    multiverse_id       INTEGER CHECK (multiverse_id >= 0 OR multiverse_id IS NULL), -- id from gatherer.wizards.com, id per lang
    power               VARCHAR(255),
    toughness           VARCHAR(255),
    card_id             INTEGER REFERENCES card (id) ON DELETE CASCADE
);

CREATE INDEX idx_card_face_name_card_id on card_face(card_id, name);

CREATE TABLE card_translation
(
    id            INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          VARCHAR(255) NOT NULL CHECK ( name <> '' ),
    multiverse_id INTEGER CHECK (multiverse_id >= 0 OR multiverse_id IS NULL),
    text          VARCHAR(800),
    flavor_text   VARCHAR(500),
    type_line     VARCHAR(255),
    lang_lang     CHAR(3) REFERENCES lang (lang),
    face_id       INTEGER REFERENCES card_face (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, face_id)
);

CREATE TABLE face_super_type
(
    face_id INTEGER REFERENCES card_face (id),
    type_id INTEGER REFERENCES super_type (id),
    UNIQUE (face_id, type_id)
);

CREATE TABLE face_sub_type
(
    face_id INTEGER REFERENCES card_face (id),
    type_id INTEGER REFERENCES sub_type (id),
    UNIQUE (face_id, type_id)
);

CREATE TABLE face_card_type
(
    face_id INTEGER REFERENCES card_face (id),
    type_id INTEGER REFERENCES card_type (id),
    UNIQUE (face_id, type_id)
);


CREATE TABLE card_image
(
    id         INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    image_path VARCHAR(255) NOT NULL CHECK ( image_path <> '' ),
    card_id    INTEGER      NOT NULL CHECK (card_id >= 0),
    face_id    INTEGER,
    mime_type  VARCHAR(100) NOT NULL CHECK (mime_type <> ''),
    phash1     BIT(64),
    phash2     BIT(64),
    phash3     BIT(64),
    phash4     BIT(64),
    lang_lang  CHAR(3) REFERENCES lang (lang),
    UNIQUE (image_path)
);

CREATE TABLE card_collection
(
    id      INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    card_id INTEGER      NOT NULL CHECK (card_id >= 0),
    user_id VARCHAR(100) NOT NULL CHECK (user_id <> ''),
    amount  INTEGER      NOT NULL DEFAULT 0 CHECK (amount >= 0 AND amount < 1000),
    UNIQUE (card_id, user_id)
);
