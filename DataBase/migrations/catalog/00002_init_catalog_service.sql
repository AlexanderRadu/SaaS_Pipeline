-- +goose Up
CREATE TABLE IF NOT EXISTS services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(120) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    docker_image TEXT NOT NULL UNIQUE,
    exposed_port TEXT NOT NULL,
    scheme TEXT NOT NULL,
    default_user TEXT NOT NULL
    );
INSERT INTO services (name, description, docker_image, exposed_port, scheme, default_user) VALUES
    ('Ubuntu CLI', 'Терминал Ubuntu 22.04', 'ubuntu:22.04', '22/tcp', 'ssh', 'root'),
    ('PostgreSQL', 'База данных Postgres 15', 'postgres:15-alpine', '5432/tcp', 'postgres', 'postgres')
    ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS services;