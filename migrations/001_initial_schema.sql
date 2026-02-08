-- PostgreSQL schema for acnil-bot audit system
-- This schema replaces Google Sheets with a relational database

-- Games table (replaces "Juegos de mesa" sheet)
CREATE TABLE IF NOT EXISTS games (
    id VARCHAR(20) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(100),
    holder VARCHAR(100),
    comments TEXT,
    take_date TIMESTAMP WITH TIME ZONE,
    return_date TIMESTAMP WITH TIME ZONE,
    price VARCHAR(50),
    publisher VARCHAR(100),
    bgg VARCHAR(50),
    avg_rate DECIMAL(3,2),
    avg_weight DECIMAL(3,2),
    age INTEGER,
    min_players INTEGER,
    max_players INTEGER,
    playingtime DECIMAL(6,2),
    yearpublished INTEGER,
    language_dependence VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Audit entries table (replaces "Audit" sheet)
CREATE TABLE IF NOT EXISTS audit_entries (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('new', 'removed', 'update')),
    game_id VARCHAR(20) NOT NULL,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(100),
    holder VARCHAR(100),
    comments TEXT,
    take_date TIMESTAMP WITH TIME ZONE,
    return_date TIMESTAMP WITH TIME ZONE,
    price VARCHAR(50),
    publisher VARCHAR(100),
    bgg VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Members table (replaces "Miembros Telegram" sheet)
CREATE TABLE IF NOT EXISTS members (
    id SERIAL PRIMARY KEY,
    nickname VARCHAR(100) NOT NULL UNIQUE,
    telegram_id VARCHAR(50) NOT NULL UNIQUE,
    permissions VARCHAR(20) NOT NULL CHECK (permissions IN ('no', 'si', 'admin')),
    state_action VARCHAR(100),
    state_data TEXT,
    telegram_name VARCHAR(255),
    telegram_username VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_games_holder ON games(holder);
CREATE INDEX IF NOT EXISTS idx_games_location ON games(location);
CREATE INDEX IF NOT EXISTS idx_games_name ON games(name);
CREATE INDEX IF NOT EXISTS idx_audit_entries_timestamp ON audit_entries(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_entries_game_id ON audit_entries(game_id);
CREATE INDEX IF NOT EXISTS idx_audit_entries_type ON audit_entries(type);
CREATE INDEX IF NOT EXISTS idx_members_telegram_id ON members(telegram_id);
CREATE INDEX IF NOT EXISTS idx_members_permissions ON members(permissions);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_games_updated_at BEFORE UPDATE ON games
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_members_updated_at BEFORE UPDATE ON members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();