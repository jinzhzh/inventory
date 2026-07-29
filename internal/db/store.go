package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// SKU 商品SKU
type SKU struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	SKUCode     string    `json:"sku_code"`
	Category    string    `json:"category"`
	Unit        string    `json:"unit"`
	AlertQty    int       `json:"alert_qty"`
	CostPrice   float64   `json:"cost_price"`
	SalePrice   float64   `json:"sale_price"`
	CurrentQty  int       `json:"current_qty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Transaction 出入库记录
type Transaction struct {
	ID        string    `json:"id"`
	SKU       string    `json:"sku"`
	SKUName   string    `json:"sku_name"`
	Type      string    `json:"type"` // inbound / outbound
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	Total     float64   `json:"total"`
	Remark    string    `json:"remark"`
	Operator  string    `json:"operator"`
	CreatedAt time.Time `json:"created_at"`
}

// Inventory 库存快照
type Inventory struct {
	SKU        string  `json:"sku"`
	Name       string  `json:"name"`
	SKUCode    string  `json:"sku_code"`
	CurrentQty int     `json:"current_qty"`
	AlertQty   int     `json:"alert_qty"`
	Category   string  `json:"category"`
	Unit       string  `json:"unit"`
}

// Alert 低库存预警
type Alert struct {
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	SKUCode    string `json:"sku_code"`
	CurrentQty int    `json:"current_qty"`
	AlertQty   int    `json:"alert_qty"`
	Unit       string `json:"unit"`
}

// Store 数据库操作
type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	dsn := path + "?_journal_mode=WAL&_txlock=immediate"
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1)
	return &Store{db: conn}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Init() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS skus (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			sku_code TEXT UNIQUE NOT NULL,
			category TEXT DEFAULT '',
			unit TEXT DEFAULT '个',
			alert_qty INTEGER DEFAULT 10,
			cost_price REAL DEFAULT 0,
			sale_price REAL DEFAULT 0,
			current_qty INTEGER DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			sku TEXT NOT NULL REFERENCES skus(id),
			type TEXT NOT NULL CHECK(type IN ('inbound','outbound')),
			quantity INTEGER NOT NULL,
			price REAL DEFAULT 0,
			total REAL DEFAULT 0,
			remark TEXT DEFAULT '',
			operator TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tx_sku ON transactions(sku)`,
		`CREATE INDEX IF NOT EXISTS idx_tx_created ON transactions(created_at DESC)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("建表失败 [%s]: %w", stmt[:30], err)
		}
	}
	return nil
}

// ---- SKU ----

func (s *Store) CreateSKU(ctx context.Context, sku SKU) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO skus (id, name, sku_code, category, unit, alert_qty, cost_price, sale_price) VALUES (?,?,?,?,?,?,?,?)`,
		sku.ID, sku.Name, sku.SKUCode, sku.Category, sku.Unit, sku.AlertQty, sku.CostPrice, sku.SalePrice,
	)
	return err
}

func (s *Store) UpdateSKU(ctx context.Context, sku SKU) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE skus SET name=?,sku_code=?,category=?,unit=?,alert_qty=?,cost_price=?,sale_price=?,updated_at=datetime('now') WHERE id=?`,
		sku.Name, sku.SKUCode, sku.Category, sku.Unit, sku.AlertQty, sku.CostPrice, sku.SalePrice, sku.ID,
	)
	return err
}

func (s *Store) DeleteSKU(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM skus WHERE id=?`, id)
	return err
}

func (s *Store) GetSKU(ctx context.Context, id string) (*SKU, error) {
	sku := &SKU{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id,name,sku_code,category,unit,alert_qty,cost_price,sale_price,current_qty,created_at,updated_at FROM skus WHERE id=?`,
		id).Scan(&sku.ID, &sku.Name, &sku.SKUCode, &sku.Category, &sku.Unit, &sku.AlertQty, &sku.CostPrice, &sku.SalePrice, &sku.CurrentQty, &sku.CreatedAt, &sku.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return sku, nil
}

func (s *Store) ListSKUs(ctx context.Context) ([]SKU, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id,name,sku_code,category,unit,alert_qty,cost_price,sale_price,current_qty,created_at,updated_at FROM skus ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var skus []SKU
	for rows.Next() {
		var s SKU
		if err := rows.Scan(&s.ID, &s.Name, &s.SKUCode, &s.Category, &s.Unit, &s.AlertQty, &s.CostPrice, &s.SalePrice, &s.CurrentQty, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		skus = append(skus, s)
	}
	return skus, rows.Err()
}

// ---- Transaction ----

func (s *Store) AddTransaction(ctx context.Context, tx Transaction) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO transactions (id, sku, type, quantity, price, total, remark, operator) VALUES (?,?,?,?,?,?,?,?)`,
		tx.ID, tx.SKU, tx.Type, tx.Quantity, tx.Price, tx.Total, tx.Remark, tx.Operator,
	)
	return err
}

func (s *Store) Ledger(ctx context.Context, limit, offset int) ([]Transaction, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT t.id, t.sku, s.name, t.type, t.quantity, t.price, t.total, t.remark, t.operator, t.created_at
		 FROM transactions t JOIN skus s ON t.sku=s.id
		 ORDER BY t.created_at DESC LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var txs []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.SKU, &t.SKUName, &t.Type, &t.Quantity, &t.Price, &t.Total, &t.Remark, &t.Operator, &t.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}

// ---- Inventory ----

func (s *Store) Inventory(ctx context.Context) ([]Inventory, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, sku_code, current_qty, alert_qty, category, unit FROM skus ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var inv []Inventory
	for rows.Next() {
		var i Inventory
		if err := rows.Scan(&i.SKU, &i.Name, &i.SKUCode, &i.CurrentQty, &i.AlertQty, &i.Category, &i.Unit); err != nil {
			return nil, err
		}
		inv = append(inv, i)
	}
	return inv, rows.Err()
}

// ---- Alerts ----

func (s *Store) Alerts(ctx context.Context) ([]Alert, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, sku_code, current_qty, alert_qty, unit FROM skus WHERE current_qty <= alert_qty ORDER BY current_qty ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.SKU, &a.Name, &a.SKUCode, &a.CurrentQty, &a.AlertQty, &a.Unit); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

// ---- Adjust stock ----

func (s *Store) AdjustStock(ctx context.Context, skuID string, delta int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE skus SET current_qty = current_qty + ?, updated_at = datetime('now') WHERE id = ?`,
		delta, skuID)
	return err
}

// ---- Stats ----

func (s *Store) Stats(ctx context.Context) (map[string]any, error) {
	type row struct {
		totalSKU    int
		totalIn     int
		totalOut    int
		alertCount  int
		totalValue  float64
	}
	var r row
	err := s.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM skus),
			(SELECT COALESCE(SUM(quantity),0) FROM transactions WHERE type='inbound'),
			(SELECT COALESCE(SUM(quantity),0) FROM transactions WHERE type='outbound'),
			(SELECT COUNT(*) FROM skus WHERE current_qty <= alert_qty),
			(SELECT COALESCE(SUM(current_qty * sale_price),0) FROM skus)
	`).Scan(&r.totalSKU, &r.totalIn, &r.totalOut, &r.alertCount, &r.totalValue)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"total_sku":    r.totalSKU,
		"total_in":     r.totalIn,
		"total_out":    r.totalOut,
		"alert_count":  r.alertCount,
		"total_value":  r.totalValue,
	}, nil
}