CREATE TABLE IF NOT EXISTS "VkUsers" (
        id			INTEGER NOT NULL PRIMARY KEY UNIQUE,
        FirstName	TEXT NOT NULL,
        LastName	TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS "TgUsers" (
        id			INTEGER NOT NULL PRIMARY KEY UNIQUE,
        FirstName	TEXT NOT NULL,
        LastName	TEXT,
        Username	TEXT

);

CREATE TABLE IF NOT EXISTS "Users" (
        id			INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
        vkID		INTEGER UNIQUE,
        tgID		INTEGER UNIQUE,

        FOREIGN KEY("vkID") REFERENCES VkUsers("id"),
        FOREIGN KEY("tgID") REFERENCES "TgUsers"("id")
);

CREATE TABLE IF NOT EXISTS "Admins" (
        id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
        username    TEXT NOT NULL UNIQUE,
        passhash    BLOB NOT NULL,
        name        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS "DishKinds" (
        id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
        repr        TEXT NOT NULL UNIQUE,
        price       INTEGER NOT NULL
);

INSERT OR REPLACE INTO DishKinds (repr, price) VALUES
        ('корм', 185),
        ('напиток', 40),
        ('суп', 75);

CREATE TABLE IF NOT EXISTS "Dishes" (
        id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
        name        TEXT NOT NULL,
        description TEXT,
        quantity    INTEGER,
        kind        INTEGER NOT NULL,

        FOREIGN KEY ("kind") REFERENCES DishKinds("id")
);

CREATE TABLE IF NOT EXISTS "Orders" (
        id			INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
        UID			INTEGER NOT NULL,
        time		INTEGER NOT NULL,
        offer_id    INTEGER DEFAULT 0,

        FOREIGN KEY("offer_id") REFERENCES Offers("id") ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS "OrderItems" (
        order_id       INTEGER NOT NULL,
        dish_id        INTEGER NOT NULL,
        quantity       INTEGER NOT NULL,

        PRIMARY KEY("order_id", "dish_id"),
        FOREIGN KEY("order_id") REFERENCES Orders("id") ON DELETE CASCADE,
        FOREIGN KEY("dish_id") REFERENCES Dishes("id") ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS "Offers" (
        id              INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
        name            TEXT,
        price           INTEGER NOT NULL,
        expires         INTEGER
);

CREATE TABLE IF NOT EXISTS "OfferItems" (
        offer_id        INTEGER NOT NULL,
        kind_id         INTEGER NOT NULL,
        quantity        INTEGER NOT NULL,

        FOREIGN KEY("offer_id") REFERENCES Orders("id") ON DELETE CASCADE,
        FOREIGN KEY("kind_id") REFERENCES DishKinds("id"),
        PRIMARY KEY ("offer_id", "kind_id")
);