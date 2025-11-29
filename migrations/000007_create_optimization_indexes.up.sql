-- Additional optimization indexes for better query performance

-- Composite indexes for common query patterns

-- For finding users by email verification status
CREATE INDEX idx_users_email_verified ON users(email_verified) WHERE email_verified = TRUE;

-- For finding recent logins
CREATE INDEX idx_users_last_login_at ON users(last_login_at DESC) WHERE last_login_at IS NOT NULL;

-- For organizations by creation date (most recent first)
CREATE INDEX idx_organizations_created_at_desc ON organizations(created_at DESC);

-- For finding user roles by role type
CREATE INDEX idx_user_org_roles_role ON user_org_roles(role);

-- For projects by organization and creation date
CREATE INDEX idx_projects_org_created_at ON projects(organization_id, created_at DESC);

-- For active API keys (not expired) - use a simple check for non-null expiration
CREATE INDEX idx_api_keys_active ON api_keys(expires_at) WHERE expires_at IS NOT NULL;

-- For recently used API keys
CREATE INDEX idx_api_keys_last_used ON api_keys(last_used_at DESC) WHERE last_used_at IS NOT NULL;

-- For API key lookups by prefix (for UI display)
CREATE INDEX idx_api_keys_prefix_active ON api_keys(key_prefix) WHERE expires_at IS NOT NULL;

-- JSONB indexes for settings and metadata

-- GIN index for organization settings (for querying specific config values)
CREATE INDEX idx_organizations_settings ON organizations USING GIN(settings);

-- GIN index for project settings
CREATE INDEX idx_projects_settings ON projects USING GIN(settings);

-- GIN index for API key permissions (for checking specific permissions)
CREATE INDEX idx_api_keys_permissions ON api_keys USING GIN(permissions);

-- GIN index for API key restrictions
CREATE INDEX idx_api_keys_restrictions ON api_keys USING GIN(restrictions);

-- Partial indexes for common filtered queries

-- For unverified users (cleanup, notifications)
CREATE INDEX idx_users_unverified ON users(created_at) WHERE email_verified = FALSE;

-- For expired API keys (cleanup) - Note: application should filter with current time
CREATE INDEX idx_api_keys_expired_cleanup ON api_keys(expires_at) WHERE expires_at IS NOT NULL;

-- For API keys never used (analytics)
CREATE INDEX idx_api_keys_never_used ON api_keys(created_at) WHERE last_used_at IS NULL;