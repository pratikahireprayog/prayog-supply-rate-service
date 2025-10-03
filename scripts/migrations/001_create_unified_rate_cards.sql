-- Migration: Create unified_rate_cards table
-- Created: October 1, 2025
-- Description: Creates the unified_rate_cards table for storing rate card configurations

-- Create unified_rate_cards table
CREATE TABLE IF NOT EXISTS unified_rate_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    product_type VARCHAR(50) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_default BOOLEAN NOT NULL DEFAULT false,
    effective_from TIMESTAMP NOT NULL,
    effective_to TIMESTAMP,
    unified_rate_card_id VARCHAR(100) NOT NULL,
    tenant_id VARCHAR(100) NOT NULL,
    api_key VARCHAR(255) NOT NULL,
    config JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(100),
    updated_by VARCHAR(100)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_unified_rate_cards_partner_code ON unified_rate_cards(partner_code);
CREATE INDEX IF NOT EXISTS idx_unified_rate_cards_unified_id ON unified_rate_cards(unified_rate_card_id);
CREATE INDEX IF NOT EXISTS idx_unified_rate_cards_tenant_id ON unified_rate_cards(tenant_id);
CREATE INDEX IF NOT EXISTS idx_unified_rate_cards_effective_dates ON unified_rate_cards(effective_from, effective_to);
CREATE INDEX IF NOT EXISTS idx_unified_rate_cards_active_default ON unified_rate_cards(is_active, is_default);

-- Add constraints
ALTER TABLE unified_rate_cards 
    ADD CONSTRAINT chk_effective_dates 
    CHECK (effective_to IS NULL OR effective_to > effective_from);

-- Create trigger function for updated_at
CREATE OR REPLACE FUNCTION update_unified_rate_cards_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
DROP TRIGGER IF EXISTS trg_unified_rate_cards_updated_at ON unified_rate_cards;
CREATE TRIGGER trg_unified_rate_cards_updated_at
    BEFORE UPDATE ON unified_rate_cards
    FOR EACH ROW
    EXECUTE FUNCTION update_unified_rate_cards_updated_at();

-- Add table comment
COMMENT ON TABLE unified_rate_cards IS 'Stores unified rate card configurations for partners';
COMMENT ON COLUMN unified_rate_cards.partner_code IS 'Partner identifier code (e.g., delhivery, porter, unified)';
COMMENT ON COLUMN unified_rate_cards.unified_rate_card_id IS 'ID of the rate card in the unified rate API';
COMMENT ON COLUMN unified_rate_cards.tenant_id IS 'Tenant ID for the unified rate API';
COMMENT ON COLUMN unified_rate_cards.api_key IS 'API key for accessing the unified rate API';
COMMENT ON COLUMN unified_rate_cards.is_default IS 'Whether this is the default rate card for the partner';
COMMENT ON COLUMN unified_rate_cards.effective_from IS 'Date from which this rate card is valid';
COMMENT ON COLUMN unified_rate_cards.effective_to IS 'Date until which this rate card is valid (NULL = no end date)';


