-- Reverse migration (DOWN) for 001_Initial

-- Remove migration record
DELETE FROM migrations WHERE name = '001_Initial';

-- Drop Component Log Table
DROP TABLE IF EXISTS component_logs;

-- Drop Components Table
DROP TABLE IF EXISTS components;

-- Drop gpio_type Enum
DROP TYPE IF EXISTS gpio_type;

-- Drop Controllers Table
DROP TABLE IF EXISTS controllers;

-- Drop User-Zones Permissions Table
DROP TABLE IF EXISTS user_zones_permissions;

-- Drop access_level Enum
DROP TYPE IF EXISTS access_level;

-- Drop Zones Table
DROP TABLE IF EXISTS zones;

-- Drop Floors Table
DROP TABLE IF EXISTS floors;

-- Drop Users Table
DROP TABLE IF EXISTS users;

-- Drop AI Logs Table
DROP TABLE IF EXISTS ai_logs;

-- Drop Migration Table
DROP TABLE IF EXISTS migration;
