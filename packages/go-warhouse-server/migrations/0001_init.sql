-- 0001_init.sql: initial warehouse-server schema (PLAN.md §5).
-- Forward-only. New changes go in 0002_*.sql, never edit this file.
-- Times are UTC RFC3339 TEXT. Quantities are INTEGER units.

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    display_name TEXT,
    created_at TEXT NOT NULL,
    disabled_at TEXT
);

CREATE TABLE api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users (id),
    key_hash TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    permission TEXT NOT NULL CHECK (permission IN ('read', 'read-movements', 'write', 'admin')),
    created_at TEXT NOT NULL,
    last_used_at TEXT,
    revoked_at TEXT
);

CREATE TABLE vendors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE warehouses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    address TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE locations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    warehouse_id INTEGER NOT NULL REFERENCES warehouses (id),
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    UNIQUE (warehouse_id, code)
);

CREATE TABLE items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sku TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    unit TEXT NOT NULL DEFAULT 'pcs',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE vendor_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    vendor_id INTEGER NOT NULL REFERENCES vendors (id),
    item_id INTEGER NOT NULL REFERENCES items (id),
    vendor_sku TEXT NOT NULL,
    vendor_name TEXT,
    UNIQUE (vendor_id, vendor_sku)
);

CREATE TABLE stock (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    vendor_item_id INTEGER NOT NULL REFERENCES vendor_items (id),
    warehouse_id INTEGER NOT NULL REFERENCES warehouses (id),
    location_id INTEGER NOT NULL REFERENCES locations (id),
    quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    UNIQUE (vendor_item_id, warehouse_id, location_id)
);

-- One logical movement per business operation. Transfers carry both
-- from_location_id and to_location_id in a single row (PLAN.md §5).
-- stock_id is the affected row for immediate non-transfer effects;
-- NULL for planned movements and transfers.
CREATE TABLE stock_movements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    stock_id INTEGER REFERENCES stock (id),
    vendor_item_id INTEGER NOT NULL REFERENCES vendor_items (id),
    warehouse_id INTEGER NOT NULL REFERENCES warehouses (id),
    from_location_id INTEGER REFERENCES locations (id),
    to_location_id INTEGER REFERENCES locations (id),
    type TEXT NOT NULL CHECK (type IN ('receive', 'remove', 'transfer', 'adjust')),
    status TEXT NOT NULL DEFAULT 'executed' CHECK (status IN ('planned', 'executed', 'cancelled')),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    planned_at TEXT,
    executed_at TEXT,
    reference TEXT,
    note TEXT,
    created_by INTEGER NOT NULL REFERENCES users (id),
    created_at TEXT NOT NULL
);

CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
);

CREATE INDEX idx_api_keys_hash ON api_keys (key_hash);
CREATE INDEX idx_locations_warehouse ON locations (warehouse_id);
CREATE INDEX idx_vendor_items_item ON vendor_items (item_id);
CREATE INDEX idx_stock_lookup ON stock (vendor_item_id, warehouse_id);
CREATE INDEX idx_movements_lookup ON stock_movements (vendor_item_id, warehouse_id, status);
CREATE INDEX idx_movements_reference ON stock_movements (reference);
