-- Drop functions
DROP FUNCTION IF EXISTS set_user_context(UUID);
DROP FUNCTION IF EXISTS has_org_role(UUID, UUID, VARCHAR);

-- Drop RLS policies
DROP POLICY IF EXISTS org_members_only ON organizations;
DROP POLICY IF EXISTS project_org_access ON projects;
DROP POLICY IF EXISTS api_key_project_access ON api_keys;

-- Disable RLS on tables
ALTER TABLE organizations DISABLE ROW LEVEL SECURITY;
ALTER TABLE projects DISABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys DISABLE ROW LEVEL SECURITY;

-- Remove custom variable (optional, as it's database-wide)
-- ALTER DATABASE RESET app.current_user_id;