-- +goose Up
CREATE TABLE IF NOT EXISTS instances (
                                         id TEXT NOT NULL,
                                         user_id BIGINT NOT NULL,
                                         service_name TEXT NOT NULL,
                                         status TEXT NOT NULL,
                                         connection_url TEXT,
                                         PRIMARY KEY (id, user_id),

    CONSTRAINT valid_status CHECK (status IN ('running', 'failed', 'deleted'))
    );

CREATE INDEX IF NOT EXISTS idx_instances_user_id ON instances (user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_instances_user_id;
DROP TABLE IF EXISTS instances;
