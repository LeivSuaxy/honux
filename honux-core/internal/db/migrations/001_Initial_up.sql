-- Initial migration (UP) for 001_Initial
-- Create migration table

CREATE TABLE IF NOT EXISTS migration (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create ai_logs table

CREATE TABLE IF NOT EXISTS ai_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    prompt TEXT NOT NULL,
    result TEXT NOT NULL,
    tokens INTEGER NOT NULL DEFAULT 0,
    model VARCHAR(255),
    executed_by VARCHAR(255)
);

-- Create User Table

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE
);

-- Create Floor Table

CREATE TABLE IF NOT EXISTS floors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    name VARCHAR(255) UNIQUE NOT NULL,
    level INTEGER UNIQUE NOT NULL
);

-- Create Zone Table

CREATE TABLE IF NOT EXISTS zones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    floor_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    name VARCHAR(50) NOT NULL,
    short_identifier VARCHAR(7) UNIQUE,
    shape_type VARCHAR(50) NOT NULL,
    geometry JSONB,
    color VARCHAR(7),

    CONSTRAINT uq_floor_zone_name UNIQUE (floor_id, name)
    CONSTRAINT fk_zones_floors
        FOREIGN KEY (floor_id)
        REFERENCES floors(id)
        ON DELETE CASCADE
);

-- Create User-Zone Access Level Enum

CREATE TYPE access_level AS ENUM ('read', 'write', 'admin');

-- Create User-Zones Relation

CREATE TABLE IF NOT EXISTS user_zones_permissions(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    user_id UUID NOT NULL,
    zone_id UUID NOT NULL,
    acces_level access_level NOT NULL DEFAULT 'read',

    CONSTRAINT uq_user_zone UNIQUE (user_id, zone_id),
    CONSTRAINT fk_user_zones_permissions_users
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_user_zones_permissions_zones
        FOREIGN KEY (zone_id)
        REFERENCES zones(id)
        ON DELETE CASCADE
);

-- Create Controllers Table

CREATE TABLE IF NOT EXISTS controllers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    zone_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    induced_id VARCHAR(50) UNIQUE DEFAULT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT DEFAULT NULL,
    device_type VARCHAR(100) NOT NULL,
    last_ip_address VARCHAR(45) DEFAULT NULL,
    mqtt_topic VARCHAR(100) UNIQUE DEFAULT NULL,
    is_online BOOLEAN NOT NULL DEFAULT FALSE,
    last_ping TIMESTAMP DEFAULT NULL,
    pos_x INTEGER NOT NULL DEFAULT 0,
    pos_y INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT fk_controllers_zones
        FOREIGN KEY (zone_id)
        REFERENCES zones(id)
        ON DELETE CASCADE
);

-- Create gpio_types Enum

CREATE TYPE gpio_type AS ENUM ('digital', 'analog');

-- Create Components Table

CREATE TABLE IF NOT EXISTS components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    controller_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(100) NOT NULL,
    gpio_pin INTEGER NOT NULL,
    gpio_type gpio_type NOT NULL,
    pos_x INTEGER NOT NULL DEFAULT 0,
    pos_y INTEGER NOT NULL DEFAULT 0,
    current_state JSONB,

    CONSTRAINT uq_controller_component_name UNIQUE (controller_id, name)
    CONSTRAINT uq_controller_gpio UNIQUE (controller_id, gpio_pin)
    CONSTRAINT fk_components_controllers
        FOREIGN KEY (controller_id)
        REFERENCES controllers(id)
        ON DELETE CASCADE
);

-- Create Component Log Table

CREATE TABLE IF NOT EXISTS component_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    component_id UUID NOT NULL,
    type VARCHAR NOT NULL,
    value NUMERIC(10, 2) NOT NULL,
    unit VARCHAR,
    metadata JSONB,

    CONSTRAINT fk_component_logs_components
        FOREIGN KEY (component_id)
        REFERENCES components(id)
        ON DELETE CASCADE
);
