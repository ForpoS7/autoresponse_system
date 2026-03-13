-- Users table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- HH Tokens table
CREATE TABLE IF NOT EXISTS hh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Vacancies table
CREATE TABLE IF NOT EXISTS vacancies (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    employer VARCHAR(255),
    url VARCHAR(500) NOT NULL,
    description TEXT,
    salary_from BIGINT,
    salary_to BIGINT,
    currency VARCHAR(10),
    region VARCHAR(255),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Auto apply requests table
CREATE TABLE IF NOT EXISTS auto_apply_requests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    query VARCHAR(500) NOT NULL,
    apply_count INT NOT NULL DEFAULT 0,
    applied_count INT NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Auto apply logs table
CREATE TABLE IF NOT EXISTS auto_apply_logs (
    id BIGSERIAL PRIMARY KEY,
    request_id BIGINT NOT NULL REFERENCES auto_apply_requests(id) ON DELETE CASCADE,
    vacancy_id BIGINT NOT NULL,
    vacancy_url VARCHAR(500) NOT NULL,
    cover_letter TEXT,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_hh_tokens_user_id ON hh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_vacancies_user_id ON vacancies(user_id);
CREATE INDEX IF NOT EXISTS idx_auto_apply_requests_user_id ON auto_apply_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_auto_apply_logs_request_id ON auto_apply_logs(request_id);
