package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(path string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create data dir: %w", err)
		}
	}
	var err error
	DB, err = sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	DB.SetMaxOpenConns(1)
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	if err = createTables(); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}

func createTables() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			nickname TEXT NOT NULL DEFAULT '',
			avatar TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS brands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			logo TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			followers INTEGER NOT NULL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			icon TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			subtitle TEXT NOT NULL DEFAULT '',
			price REAL NOT NULL,
			original_price REAL NOT NULL DEFAULT 0,
			image TEXT NOT NULL DEFAULT '',
			images TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT '',
			category_id INTEGER NOT NULL DEFAULT 0,
			brand_id INTEGER NOT NULL DEFAULT 0,
			brand_name TEXT NOT NULL DEFAULT '',
			shop TEXT NOT NULL DEFAULT '',
			stock INTEGER NOT NULL DEFAULT 999,
			sales INTEGER NOT NULL DEFAULT 0,
			description TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
			is_genuine INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_products_category ON products(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_products_brand ON products(brand_id)`,
		`CREATE TABLE IF NOT EXISTS cart_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL DEFAULT 1,
			selected INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cart_user ON cart_items(user_id)`,
		`CREATE TABLE IF NOT EXISTS orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			order_no TEXT NOT NULL UNIQUE,
			total REAL NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			items_json TEXT NOT NULL,
			address TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id)`,
		`CREATE TABLE IF NOT EXISTS reviews (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			rating INTEGER NOT NULL DEFAULT 5,
			content TEXT NOT NULL DEFAULT '',
			images TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_product ON reviews(product_id)`,
		// SKU: a specific spec combination (color/size/version) of a product.
		`CREATE TABLE IF NOT EXISTS skus (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL,
			spec TEXT NOT NULL,
			spec_text TEXT NOT NULL DEFAULT '',
			price REAL NOT NULL,
			stock INTEGER NOT NULL DEFAULT 0,
			sku_code TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_skus_product ON skus(product_id)`,
		// Shipments.
		`CREATE TABLE IF NOT EXISTS shipments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			tracking_no TEXT NOT NULL,
			carrier TEXT NOT NULL DEFAULT '天猫超市配送',
			status TEXT NOT NULL DEFAULT 'shipped',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_shipments_order ON shipments(order_id)`,
		`CREATE TABLE IF NOT EXISTS shipment_tracks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			shipment_id INTEGER NOT NULL,
			description TEXT NOT NULL,
			location TEXT NOT NULL DEFAULT '',
			occurred_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tracks_shipment ON shipment_tracks(shipment_id)`,
		// Payments.
		`CREATE TABLE IF NOT EXISTS payments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			amount REAL NOT NULL,
			method TEXT NOT NULL DEFAULT 'alipay',
			transaction_no TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_order ON payments(order_id)`,
		// Refunds.
		`CREATE TABLE IF NOT EXISTS refunds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT 'refund',
			amount REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'pending',
			admin_note TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_refunds_user ON refunds(user_id)`,
		// Coupons.
		`CREATE TABLE IF NOT EXISTS coupons (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			coupon_type TEXT NOT NULL DEFAULT 'deduct',
			threshold REAL NOT NULL DEFAULT 0,
			value REAL NOT NULL DEFAULT 0,
			total_count INTEGER NOT NULL DEFAULT 0,
			claimed_count INTEGER NOT NULL DEFAULT 0,
			start_date TEXT NOT NULL DEFAULT '',
			end_date TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_coupons (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			coupon_id INTEGER NOT NULL,
			is_used INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, coupon_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_coupons_user ON user_coupons(user_id)`,
		// Favorites: products a user has favorited (wishlist).
		`CREATE TABLE IF NOT EXISTS favorites (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, product_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites(user_id)`,
		// Addresses: shipping destinations.
		`CREATE TABLE IF NOT EXISTS addresses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			phone TEXT NOT NULL,
			detail TEXT NOT NULL,
			is_default INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_addresses_user ON addresses(user_id)`,
		// FTS5.
		`CREATE VIRTUAL TABLE IF NOT EXISTS products_fts USING fts5(name, subtitle, category, tags, description, content='products', content_rowid='id')`,
		`CREATE TRIGGER IF NOT EXISTS products_ai AFTER INSERT ON products BEGIN
			INSERT INTO products_fts(rowid, name, subtitle, category, tags, description)
			VALUES (new.id, new.name, new.subtitle, new.category, new.tags, new.description);
		END`,
		`CREATE TRIGGER IF NOT EXISTS products_ad AFTER DELETE ON products BEGIN
			INSERT INTO products_fts(products_fts, rowid, name, subtitle, category, tags, description)
			VALUES ('delete', old.id, old.name, old.subtitle, old.category, old.tags, old.description);
		END`,
		`CREATE TRIGGER IF NOT EXISTS products_au AFTER UPDATE ON products BEGIN
			INSERT INTO products_fts(products_fts, rowid, name, subtitle, category, tags, description)
			VALUES ('delete', old.id, old.name, old.subtitle, old.category, old.tags, old.description);
			INSERT INTO products_fts(rowid, name, subtitle, category, tags, description)
			VALUES (new.id, new.name, new.subtitle, new.category, new.tags, new.description);
		END`,
	}
	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			return fmt.Errorf("exec: %w", err)
		}
	}
	return migrate()
}

// migrate applies additive schema changes for databases created before a
// feature shipped. Each step is best-effort and idempotent (errors from a
// duplicate add-column are ignored).
func migrate() error {
	// Add phone column to users (added after launch).
	_, _ = DB.Exec(`ALTER TABLE users ADD COLUMN phone TEXT NOT NULL DEFAULT ''`)
	return nil
}
