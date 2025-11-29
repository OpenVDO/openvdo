-- Drop trigger
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;

-- Drop indexes
DROP INDEX IF EXISTS idx_organizations_name;
DROP INDEX IF EXISTS idx_organizations_created_at;

-- Drop organizations table
DROP TABLE IF EXISTS organizations;