-- Initialize database for OpenVDO
-- This script runs when the PostgreSQL container starts for the first time

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create additional indexes or initial data here if needed