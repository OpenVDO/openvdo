-- Drop trigger
DROP TRIGGER IF EXISTS update_api_keys_updated_at ON api_keys;

-- Drop indexes
DROP INDEX IF EXISTS idx_api_keys_project_id;
DROP INDEX IF EXISTS idx_api_keys_hash;
DROP INDEX IF EXISTS idx_api_keys_prefix;
DROP INDEX IF EXISTS idx_api_keys_expires_at;

-- Drop api_keys table
DROP TABLE IF EXISTS api_keys;