CREATE TYPE record_type AS ENUM ('income', 'expense');

CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type record_type NOT NULL,
    user_id BIGINT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_category_user FOREIGN KEY (user_id) REFERENCES users(id)
);
