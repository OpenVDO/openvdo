-- Drop trigger
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;

-- Drop indexes
DROP INDEX IF EXISTS idx_projects_org_id;
DROP INDEX IF EXISTS idx_projects_name;
DROP INDEX IF EXISTS idx_projects_created_at;

-- Drop projects table
DROP TABLE IF EXISTS projects;