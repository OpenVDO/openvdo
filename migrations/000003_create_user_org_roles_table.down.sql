-- Drop trigger
DROP TRIGGER IF EXISTS update_user_org_roles_updated_at ON user_org_roles;

-- Drop indexes
DROP INDEX IF EXISTS idx_user_org_roles_user_id;
DROP INDEX IF EXISTS idx_user_org_roles_org_id;
DROP INDEX IF EXISTS idx_user_org_roles_composite;

-- Drop user_org_roles table
DROP TABLE IF EXISTS user_org_roles;