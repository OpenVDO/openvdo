-- Create organizations table for multi-tenant architecture
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT organizations_name_unique UNIQUE (name)
);

-- Create indexes for organizations table
CREATE INDEX idx_organizations_name ON organizations(name);
CREATE INDEX idx_organizations_created_at ON organizations(created_at);

-- Add trigger for updated_at timestamp
CREATE TRIGGER update_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();