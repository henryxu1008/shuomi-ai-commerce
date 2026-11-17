CREATE TABLE IF NOT EXISTS supply_market_categories (
    id uuid PRIMARY KEY,
    parent_id uuid REFERENCES supply_market_categories(id),
    name text NOT NULL CHECK (octet_length(name) BETWEEN 1 AND 120 AND name = btrim(name)),
    enabled boolean NOT NULL,
    revision bigint NOT NULL CHECK (revision > 0),
    CHECK (parent_id IS DISTINCT FROM id)
);
CREATE UNIQUE INDEX IF NOT EXISTS supply_market_category_root_name
    ON supply_market_categories (name) WHERE parent_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS supply_market_category_child_name
    ON supply_market_categories (parent_id, name) WHERE parent_id IS NOT NULL;
ALTER TABLE supply_market_records ADD COLUMN IF NOT EXISTS category_id uuid
    REFERENCES supply_market_categories(id);
ALTER TABLE supply_market_releases ADD COLUMN IF NOT EXISTS category_id uuid
    REFERENCES supply_market_categories(id);
CREATE INDEX IF NOT EXISTS supply_market_release_category
    ON supply_market_releases (category_id, channel, id) WHERE active AND category_id IS NOT NULL;
