#!/bin/bash

# Script to create admin user and organization
# Usage: ./scripts/onboard-admin.sh

echo "Setting up initial super admin user..."
echo "Note: Make sure database migrations have been run (make migrate-up)"
echo ""

# Read user input
read -p "Enter admin email: " email
read -s -p "Enter admin password: " password
echo ""
read -p "Enter admin name: " name
read -p "Enter organization name: " org_name
read -p "Enter organization description (optional): " org_desc

echo ""
echo "Creating admin user and organization..."

# Database connection string
DB_CONN="postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}"

# Execute SQL commands
psql "$DB_CONN" << EOF
BEGIN;
-- Create admin user
INSERT INTO users (email, password_hash, name, email_verified)
VALUES ('$email', crypt('$password', gen_salt('bf')), '$name', TRUE);

-- Create organization
INSERT INTO organizations (name, description)
VALUES ('$org_name', '$org_desc');

-- Assign admin as owner
INSERT INTO user_org_roles (user_id, organization_id, role)
VALUES (
    (SELECT id FROM users WHERE email = '$email'),
    (SELECT id FROM organizations WHERE name = '$org_name'),
    'owner'
);
COMMIT;
EOF

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Admin user onboarded successfully!"
    echo "   Email: $email"
    echo "   Name: $name"
    echo "   Organization: $org_name"
    echo ""
    echo "You can now use these credentials to log in to your application."
else
    echo "❌ Failed to create admin user"
    exit 1
fi