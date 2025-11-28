CREATE TABLE IF NOT EXISTS passports (
    id UUID PRIMARY KEY,
    product_category VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    manufacturer_id VARCHAR(100) NOT NULL,
    manufacturer_name VARCHAR(255) NOT NULL,
    
    -- This is the magic column
    attributes JSONB NOT NULL, 
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    published_at TIMESTAMP WITH TIME ZONE,
    immutability_hash VARCHAR(256),
    storage_location TEXT
);

-- Create an index on the JSONB column for speed
-- Example: Quickly find all batteries with 'Chemistry' = 'LFP'
CREATE INDEX idx_passports_attributes ON passports USING gin (attributes);
CREATE INDEX idx_passports_category ON passports (product_category);