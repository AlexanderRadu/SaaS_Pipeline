-- +goose Up
CREATE TABLE IF NOT EXISTS services (
                                        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(120) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    docker_image TEXT NOT NULL UNIQUE
    );
INSERT INTO services (name, description, docker_image) VALUES
                                                           ('Ubuntu CLI', 'Терминал Ubuntu 22.04', 'ubuntu:22.04'),
                                                           ('PostgreSQL', 'База данных Postgres 15', 'postgres:15-alpine')
    ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS services;