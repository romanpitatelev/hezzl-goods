-- +migrate Up
CREATE TABLE projects
(
    id SERIAL PRIMARY KEY,
    name VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP 
);

CREATE TABLE goods
(
    id SERIAL PRIMARY KEY,
    project_id INTEGER REFERENCES projects (id) NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    priority NUMERIC NOT NULL,
    removed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO projects (name)
VALUES (
    'first entry'
);

CREATE INDEX idx_project_id ON goods(project_id);
CREATE INDEX idx_name ON goods(name);

-- +migrate Down
DROP INDEX IF EXISTS idx_name;
DROP INDEX IF EXISTS idx_project_id;
DROP TABLE goods;
DROP TABLE projects;


