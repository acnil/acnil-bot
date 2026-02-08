-- PostgreSQL schema for acnil-bot audit system (Hybrid Backend)
-- This schema only stores audit entries in PostgreSQL
-- All other data (games, members, juegatron) remains in Google Sheets

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

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_audit_entries_timestamp ON audit_entries(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_entries_game_id ON audit_entries(game_id);
CREATE INDEX IF NOT EXISTS idx_audit_entries_type ON audit_entries(type);
CREATE INDEX IF NOT EXISTS idx_audit_entries_holder ON audit_entries(holder);

-- Trigger to update created_at timestamp (optional, as we use DEFAULT NOW())
CREATE OR REPLACE FUNCTION update_created_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.created_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- This trigger is optional since we use DEFAULT NOW()
-- CREATE TRIGGER update_audit_entries_created_at BEFORE INSERT ON audit_entries
--     FOR EACH ROW EXECUTE FUNCTION update_created_at_column();