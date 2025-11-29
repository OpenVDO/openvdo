-- Create user_org_roles table for managing user permissions within organizations
CREATE TABLE user_org_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('owner', 'admin', 'developer', 'viewer')),
    invited_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE (user_id, organization_id)
);

-- Create indexes for user_org_roles table
CREATE INDEX idx_user_org_roles_user_id ON user_org_roles(user_id);
CREATE INDEX idx_user_org_roles_org_id ON user_org_roles(organization_id);
CREATE INDEX idx_user_org_roles_composite ON user_org_roles(user_id, organization_id);

-- Add trigger for updated_at timestamp
CREATE TRIGGER update_user_org_roles_updated_at
    BEFORE UPDATE ON user_org_roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();