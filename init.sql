CREATE TABLE IF NOT EXISTS user (
    id INTEGER NOT NULL,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS access_token (
    user INTEGER NOT NULL,
    token TEXT NOT NULL,
    valid_to INTEGER NOT NULL,
    PRIMARY KEY (user, token),
    FOREIGN KEY (user)
      REFERENCES user (id)
);

CREATE TABLE IF NOT EXISTS story (
    id INTEGER NOT NULL,
    title TEXT NOT NULL,
   	creator INTEGER NOT NULL,
    PRIMARY KEY (id),
    FOREIGN KEY (creator)
      REFERENCES user (id)
);
