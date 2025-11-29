-- Note: The app.current_user_id custom variable will be set by the application
-- when establishing database connections to enable RLS policies

-- Enable Row Level Security on all tenant tables
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;

-- Note: users table doesn't need RLS (global user management)
-- Note: user_org_roles doesn't need RLS (used for RLS policies)

-- Create Organizations RLS Policy
-- Users can only see organizations they belong to
CREATE POLICY org_members_only ON organizations
  FOR ALL
  USING (
    id IN (
      SELECT organization_id
      FROM user_org_roles
      WHERE user_id = current_setting('app.current_user_id', true)::uuid
    )
  );

-- Create Projects RLS Policy
-- Users can only see projects from their organizations
CREATE POLICY project_org_access ON projects
  FOR ALL
  USING (
    organization_id IN (
      SELECT organization_id
      FROM user_org_roles
      WHERE user_id = current_setting('app.current_user_id', true)::uuid
    )
  );

-- Create API Keys RLS Policy
-- Users can only see API keys from their organization's projects
CREATE POLICY api_key_project_access ON api_keys
  FOR ALL
  USING (
    project_id IN (
      SELECT id FROM projects
      WHERE organization_id IN (
        SELECT organization_id
        FROM user_org_roles
        WHERE user_id = current_setting('app.current_user_id', true)::uuid
      )
    )
  );

-- Create a helper function to check if user has specific role in organization
CREATE OR REPLACE FUNCTION has_org_role(
    p_user_id UUID,
    p_organization_id UUID,
    p_role VARCHAR
) RETURNS BOOLEAN AS '
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM user_org_roles
        WHERE user_id = p_user_id
          AND organization_id = p_organization_id
          AND (p_role IS NULL OR role = p_role)
    );
END;
' LANGUAGE plpgsql SECURITY DEFINER;

-- Create a function to set user context safely
CREATE OR REPLACE FUNCTION set_user_context(p_user_id UUID) RETURNS VOID AS '
BEGIN
    PERFORM set_config(''app.current_user_id'', p_user_id::text, true);
END;
' LANGUAGE plpgsql SECURITY DEFINER;