DROP INDEX IF EXISTS wasmorph.idx_users_email;
ALTER TABLE wasmorph.users DROP COLUMN IF EXISTS email;

