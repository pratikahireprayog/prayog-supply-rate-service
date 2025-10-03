-- Rollback: Drop unified_rate_cards table
-- Description: Removes the unified_rate_cards table and related objects

-- Drop trigger
DROP TRIGGER IF EXISTS trg_unified_rate_cards_updated_at ON unified_rate_cards;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_unified_rate_cards_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_unified_rate_cards_partner_code;
DROP INDEX IF EXISTS idx_unified_rate_cards_unified_id;
DROP INDEX IF EXISTS idx_unified_rate_cards_tenant_id;
DROP INDEX IF EXISTS idx_unified_rate_cards_effective_dates;
DROP INDEX IF EXISTS idx_unified_rate_cards_active_default;

-- Drop table
DROP TABLE IF EXISTS unified_rate_cards;


