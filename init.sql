DROP TABLE IF EXISTS user;
CREATE TABLE IF NOT EXISTS user (
    id INTEGER NOT NULL,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    PRIMARY KEY (id)
);

DROP TABLE IF EXISTS access_token;
CREATE TABLE IF NOT EXISTS access_token (
    user_id INTEGER NOT NULL,
    token TEXT NOT NULL,
    valid_to INTEGER NOT NULL,
    PRIMARY KEY (user_id, token),
    FOREIGN KEY (user_id)
      REFERENCES user (id)
);

DROP TABLE IF EXISTS story;
CREATE TABLE IF NOT EXISTS story (
    id INTEGER NOT NULL,
    title TEXT,
    description TEXT,
    start_time INTEGER,
   	creator_id INTEGER NOT NULL,
    status INTEGER,
    PRIMARY KEY (id),
    FOREIGN KEY (creator_id)
      REFERENCES user (id)
);

DROP TABLE IF EXISTS task;
CREATE TABLE IF NOT EXISTS task (
    id INTEGER NOT NULL,
    story_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    slots INTEGER DEFAULT 1,
    PRIMARY KEY (id),
    FOREIGN KEY (story_id)
      REFERENCES story (id)
);

DROP TABLE IF EXISTS assignment;
CREATE TABLE IF NOT EXISTS assignment (
    id INTEGER NOT NULL,
    task_id INTEGER NOT NULL,
    assignee_id INTEGER,
    PRIMARY KEY (id)
    FOREIGN KEY (task_id)
      REFERENCES task (id),
    FOREIGN KEY (assignee_id)
      REFERENCES user (id)
);
