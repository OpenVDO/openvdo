-- Drop optimization indexes

-- Drop single-column indexes
DROP INDEX IF EXISTS idx_users_email_verified;
DROP INDEX IF EXISTS idx_users_last_login_at;
DROP INDEX IF EXISTS idx_organizations_created_at_desc;
DROP INDEX IF EXISTS idx_user_org_roles_role;
DROP INDEX IF EXISTS idx_projects_org_created_at;
DROP INDEX IF EXISTS idx_api_keys_active;
DROP INDEX IF EXISTS idx_api_keys_last_used;
DROP INDEX IF EXISTS idx_api_keys_prefix_active;

-- Drop JSONB GIN indexes
DROP INDEX IF EXISTS idx_organizations_settings;
DROP INDEX IF EXISTS idx_projects_settings;
DROP INDEX IF EXISTS idx_api_keys_permissions;
DROP INDEX IF EXISTS idx_api_keys_restrictions;

-- Drop partial indexes
DROP INDEX IF EXISTS idx_users_unverified;
DROP INDEX IF EXISTS idx_api_keys_expired;
DROP INDEX IF EXISTS idx_api_keys_never_used;