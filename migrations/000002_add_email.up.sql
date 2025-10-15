ALTER TABLE wasmorph.users ADD COLUMN email VARCHAR(255) UNIQUE;
CREATE INDEX idx_users_email ON wasmorph.users(email);

