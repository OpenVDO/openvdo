-- SQL script to create admin user and organization
-- Variables will be substituted by the shell script

INSERT INTO users (email, password_hash, name, email_verified)
VALUES (:email, crypt(:password, gen_salt('bf')), :name, TRUE)
RETURNING id;

INSERT INTO organizations (name, description)
VALUES (:org_name, :org_desc)
RETURNING id;

INSERT INTO user_org_roles (user_id, organization_id, role)
VALUES (
    (SELECT id FROM users WHERE email = :email),
    (SELECT id FROM organizations WHERE name = :org_name),
    'owner'
);