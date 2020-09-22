CREATE TABLE IF NOT EXISTS "{{.VkUsersTable}}" (
                                         id			INTEGER NOT NULL PRIMARY KEY UNIQUE,
                                         FirstName	TEXT NOT NULL,
                                         LastName	TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS "{{.TgUsersTable}}" (
                                         id			INTEGER NOT NULL PRIMARY KEY UNIQUE,
                                         FirstName	TEXT NOT NULL,
                                         LastName	TEXT,
                                         Username	TEXT

);

CREATE TABLE IF NOT EXISTS "Users" (
                                       id			INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
                                       vkID		INTEGER UNIQUE,
                                       tgID		INTEGER UNIQUE,

                                       FOREIGN KEY("vkID") REFERENCES "{{.VkUsersTable}}"("id"),
                                       FOREIGN KEY("tgID") REFERENCES "{{.TgUsersTable}}"("id")
);

CREATE TABLE IF NOT EXISTS "Synchro" (
                                         id		INTEGER NOT NULL,
                                         fromVK	INTEGER,
                                         SyncKey	TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS "Admins" (
        id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
        username    TEXT NOT NULL UNIQUE,
        passhash    BLOB NOT NULL,
        name        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS "DishTypes" (
        id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
        description TEXT NOT NULL UNIQUE,
        price       INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS "Dishes" (
        id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
        name        TEXT NOT NULL,
        description TEXT,
        quantity    INTEGER
);



CREATE TABLE IF NOT EXISTS "Orders" (
        id			INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
        UID			INTEGER NOT NULL,
        Date		INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS "OrderItems" (
        order_id       INTEGER NOT NULL,
        dish_id        INTEGER NOT NULL,
        quantity       INTEGER NOT NULL,

        PRIMARY KEY("order_id", "dish_id"),
        FOREIGN KEY("order_id") REFERENCES Orders("id"),
        FOREIGN KEY("dish_id") REFERENCES Dishes("id") ON DELETE CASCADE
)