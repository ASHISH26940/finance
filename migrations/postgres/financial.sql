CREATE TABLE financial_records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    amount NUMERIC(15,2) NOT NULL,
    type record_type NOT NULL,
    category_id BIGINT,
    date TIMESTAMPTZ NOT NULL,
    note TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL DEFAULT NULL,
    CONSTRAINT fk_record_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_record_category FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE INDEX idx_records_user ON financial_records(user_id);
CREATE INDEX idx_records_type ON financial_records(type);
CREATE INDEX idx_records_category ON financial_records(category_id);
CREATE INDEX idx_records_date ON financial_records(date);
CREATE INDEX idx_records_deleted_at ON financial_records(deleted_at);
