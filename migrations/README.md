# Database Migrations

This directory contains the database migrations for the OpenVDO multi-tenant system.

## Migration Structure

Each migration consists of two files:
- `XXXXX_name.up.sql` - The forward migration (applies the changes)
- `XXXXX_name.down.sql` - The rollback migration (reverts the changes)

## Migration Order

1. **000001_create_users_table** - Core user authentication table
2. **000002_create_organizations_table** - Multi-tenant organization support
3. **000003_create_user_org_roles_table** - User permissions within organizations
4. **000004_create_projects_table** - Resource grouping within organizations
5. **000005_create_api_keys_table** - API authentication and access control
6. **000006_setup_rls_policies** - PostgreSQL Row Level Security for tenant isolation
7. **000007_create_optimization_indexes** - Performance optimization indexes

## Running Migrations

### Up (apply all migrations)
```bash
make migrate-up
```

### Down (rollback last migration)
```bash
make migrate-down
```

### Manual migration execution
```bash
# Using migrate tool directly
migrate -path ./migrations -database "postgres://user:pass@host:port/dbname?sslmode=disable" up
migrate -path ./migrations -database "postgres://user:pass@host:port/dbname?sslmode=disable" down 1
```

## Environment Variables

Make sure these environment variables are set before running migrations:
- `DB_USER` - Database username
- `DB_PASSWORD` - Database password
- `DB_HOST` - Database host
- `DB_PORT` - Database port
- `DB_NAME` - Database name
- `DB_SSLMODE` - SSL mode (disable, require, verify-ca, verify-full)

## Key Features

### Multi-Tenant Architecture
- **Row Level Security (RLS)** ensures complete tenant isolation
- Users can only access data from organizations they belong to
- Automatic filtering at the database level

### Security
- API keys are hashed (not stored in plain text)
- User passwords are properly hashed with bcrypt
- Role-based access control (owner, admin, developer, viewer)

### Performance
- Optimized indexes for common query patterns
- Partial indexes for filtered queries
- GIN indexes for JSONB data

## Adding New Migrations

1. Create new migration files with the next sequential number
2. Follow the naming convention: `XXXXX_descriptive_name.up/down.sql`
3. Test both up and down migrations
4. Update this README if needed

## Important Notes

- Always test migrations in a development environment first
- The `onboard-admin` make target should be used to create the initial admin user
- RLS policies provide bulletproof tenant isolation at the database level