-- Sample data for testing Prayog Supply Rate Service

-- First, insert partner configurations
INSERT INTO partners (id, name, code, type, is_active, priority, config, timeout_ms, retry_count, health_status, description, created_at, updated_at)
VALUES 
-- DHL Real-time partner
(gen_random_uuid(), 'DHL Express', 'dhl', 'real_time', true, 1, 
 '{"api_base_url": "https://express.api.dhl.com/mydhlapi/test/rates", "account_number": "533748932", "username": "shreemarut8IN", "password": "I!0pM^4sR#4nJ$1u"}',
 30000, 2, 'unknown', 'DHL Express international shipping', now(), now()),

-- Unified Rate partner
(gen_random_uuid(), 'Unified Rate Service', 'unified', 'pre_defined', true, 2, 
 '{"base_url": "https://sandbox-apis.prayog.io/gateway/ure/api", "calculate_rates_endpoint": "/external-rate-calculation/calculate"}',
 30000, 2, 'unknown', 'Prayog Unified Rate calculation service', now(), now()),

-- Delhivery partner (via unified rate)
(gen_random_uuid(), 'Delhivery', 'delhivery', 'pre_defined', true, 3, 
 '{"base_url": "https://sandbox-apis.prayog.io/gateway/ure/api", "calculate_rates_endpoint": "/external-rate-calculation/calculate"}',
 30000, 2, 'unknown', 'Delhivery domestic shipping via unified rate', now(), now()),

-- Porter partner (via unified rate)
(gen_random_uuid(), 'Porter', 'porter', 'pre_defined', true, 4, 
 '{"base_url": "https://sandbox-apis.prayog.io/gateway/ure/api", "calculate_rates_endpoint": "/external-rate-calculation/calculate"}',
 30000, 2, 'unknown', 'Porter hyperlocal delivery via unified rate', now(), now());

-- Insert unified rate card configurations
INSERT INTO unified_rate_cards (id, partner_code, name, product_type, is_active, is_default, effective_from, effective_to, unified_rate_card_id, tenant_id, api_key, config, created_at, updated_at)
VALUES 
-- Unified rate card
(gen_random_uuid(), 'unified', 'Default Unified Rate Card', 'LOGISTICS', true, true, '2025-01-01'::timestamp, '2026-12-31'::timestamp, 
 'unified_rate_card_001', '68cd38e86423698971766a14', 'prayog_live_HQT-eUnEwqJ7x4LLqo-n41WGol8w-xZG_4082bc3a',
 '{"service_types": ["standard", "express", "premium"], "supported_locations": ["domestic", "international"]}',
 now(), now()),

-- Delhivery rate card
(gen_random_uuid(), 'delhivery', 'Delhivery Standard Rate Card', 'LOGISTICS', true, true, '2025-01-01'::timestamp, '2026-12-31'::timestamp, 
 'delhivery_rate_card_001', '68cd38e86423698971766a14', 'prayog_live_HQT-eUnEwqJ7x4LLqo-n41WGol8w-xZG_4082bc3a',
 '{"service_types": ["standard", "express"], "supported_locations": ["domestic"], "coverage": "pan_india"}',
 now(), now()),

-- Porter rate card
(gen_random_uuid(), 'porter', 'Porter Hyperlocal Rate Card', 'LOGISTICS', true, true, '2025-01-01'::timestamp, '2026-12-31'::timestamp, 
 'porter_rate_card_001', '68cd38e86423698971766a14', 'prayog_live_HQT-eUnEwqJ7x4LLqo-n41WGol8w-xZG_4082bc3a',
 '{"service_types": ["standard", "express", "premium"], "supported_locations": ["local", "withinState", "metro", "roi", "specialLocation"], "coverage": "hyperlocal"}',
 now(), now());

-- Insert some sample rate entries (for historical/reference purposes)
INSERT INTO rates (id, partner_id, service_type, origin_city, dest_city, weight_min, weight_max, distance_min, distance_max, base_price, price_per_km, price_per_kg, currency, valid_from, valid_to, is_active, created_at, updated_at)
VALUES 
-- Sample rates for unified service
(gen_random_uuid(), 'unified', 'standard', '560001', '110001', 0, 1, 0, 1000, 45.50, 0.05, 15.00, 'INR', '2025-01-01'::timestamp, '2026-12-31'::timestamp, true, now(), now()),
(gen_random_uuid(), 'unified', 'express', '560001', '110001', 0, 1, 0, 1000, 65.50, 0.08, 20.00, 'INR', '2025-01-01'::timestamp, '2026-12-31'::timestamp, true, now(), now()),

-- Sample rates for delhivery
(gen_random_uuid(), 'delhivery', 'standard', '400001', '500001', 0, 2, 0, 800, 150.00, 0.10, 25.00, 'INR', '2025-01-01'::timestamp, '2026-12-31'::timestamp, true, now(), now()),

-- Sample rates for porter
(gen_random_uuid(), 'porter', 'express', '560001', '560100', 0, 1, 0, 50, 75.00, 0.15, 30.00, 'INR', '2025-01-01'::timestamp, '2026-12-31'::timestamp, true, now(), now());

