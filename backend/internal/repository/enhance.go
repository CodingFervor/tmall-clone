package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/CodingFervor/tmall-clone/backend/internal/model"
)

// ===================== SKU =====================

type SKURepo struct{ db *sql.DB }

func NewSKURepo(db *sql.DB) *SKURepo { return &SKURepo{db: db} }

func (r *SKURepo) ListByProduct(productID int64) ([]model.SKU, error) {
	rows, err := r.db.Query(
		`SELECT id, product_id, spec, spec_text, price, stock, sku_code FROM skus WHERE product_id=? ORDER BY id`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SKU{}
	for rows.Next() {
		var s model.SKU
		if err := rows.Scan(&s.ID, &s.ProductID, &s.Spec, &s.SpecText, &s.Price, &s.Stock, &s.SKUCode); err == nil {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *SKURepo) Create(s *model.SKU) error {
	res, err := r.db.Exec(
		`INSERT INTO skus (product_id, spec, spec_text, price, stock, sku_code) VALUES (?,?,?,?,?,?)`,
		s.ProductID, s.Spec, s.SpecText, s.Price, s.Stock, s.SKUCode)
	if err != nil {
		return err
	}
	s.ID, _ = res.LastInsertId()
	return nil
}

// ===================== Payment =====================

type PaymentRepo struct{ db *sql.DB }

func NewPaymentRepo(db *sql.DB) *PaymentRepo { return &PaymentRepo{db: db} }

func (r *PaymentRepo) Create(orderID, userID int64, amount float64, method string) (*model.Payment, error) {
	if method == "" {
		method = "alipay" // 天猫默认支付宝
	}
	p := &model.Payment{OrderID: orderID, UserID: userID, Amount: amount, Method: method, Status: "pending"}
	p.TransactionNo = fmt.Sprintf("TMPAY%d%d", time.Now().Unix(), userID)
	res, err := r.db.Exec(
		`INSERT INTO payments (order_id, user_id, amount, method, transaction_no, status) VALUES (?,?,?,?,?,?)`,
		p.OrderID, p.UserID, p.Amount, p.Method, p.TransactionNo, p.Status)
	if err != nil {
		return nil, err
	}
	p.ID, _ = res.LastInsertId()
	return p, nil
}

// MarkSuccess finalizes a payment + flips order to paid (transactional).
func (r *PaymentRepo) MarkSuccess(paymentID int64) (*model.Payment, error) {
	p := &model.Payment{}
	if err := r.db.QueryRow(
		`SELECT id, order_id, user_id, amount, method, transaction_no, status FROM payments WHERE id=?`, paymentID,
	).Scan(&p.ID, &p.OrderID, &p.UserID, &p.Amount, &p.Method, &p.TransactionNo, &p.Status); err != nil {
		return nil, err
	}
	if p.Status == "success" {
		return p, nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE payments SET status='success' WHERE id=?`, paymentID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE orders SET status='paid' WHERE id=?`, p.OrderID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	p.Status = "success"
	return p, nil
}

func (r *PaymentRepo) GetByOrder(orderID int64) (*model.Payment, error) {
	p := &model.Payment{}
	err := r.db.QueryRow(
		`SELECT id, order_id, user_id, amount, method, transaction_no, status FROM payments WHERE order_id=? ORDER BY id DESC LIMIT 1`, orderID,
	).Scan(&p.ID, &p.OrderID, &p.UserID, &p.Amount, &p.Method, &p.TransactionNo, &p.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// ===================== Shipment =====================

type ShipmentRepo struct{ db *sql.DB }

func NewShipmentRepo(db *sql.DB) *ShipmentRepo { return &ShipmentRepo{db: db} }

// CreateForOrder generates a shipment + initial trajectory for an order.
func (r *ShipmentRepo) CreateForOrder(orderID int64) (*model.Shipment, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	trackingNo := fmt.Sprintf("TM%s%06d", time.Now().Format("20060102"), orderID%1000000)
	res, err := tx.Exec(
		`INSERT INTO shipments (order_id, tracking_no, carrier, status) VALUES (?,?,?,?)`,
		orderID, trackingNo, "天猫超市配送", "shipped")
	if err != nil {
		return nil, err
	}
	shipID, _ := res.LastInsertId()
	seedTracks := []struct{ desc, loc string }{
		{"【天猫超市】您的订单已出库，正在打包", "天猫超市仓储中心"},
		{"【天猫超市】快件已出库，等待揽收", "天猫超市仓储中心"},
		{"【菜鸟驿站】快件已到达【杭州转运中心】", "杭州转运中心"},
		{"【菜鸟驿站】快件已发往【上海配送站】", "杭州→上海"},
	}
	now := time.Now().Add(-24 * time.Hour)
	for i, t := range seedTracks {
		_, _ = tx.Exec(
			`INSERT INTO shipment_tracks (shipment_id, description, location, occurred_at) VALUES (?,?,?,?)`,
			shipID, t.desc, t.loc, now.Add(time.Duration(i)*6*time.Hour))
	}
	if _, err := tx.Exec(`UPDATE orders SET status='shipped' WHERE id=?`, orderID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByOrder(orderID)
}

func (r *ShipmentRepo) GetByOrder(orderID int64) (*model.Shipment, error) {
	s := &model.Shipment{}
	err := r.db.QueryRow(
		`SELECT id, order_id, tracking_no, carrier, status, created_at FROM shipments WHERE order_id=?`, orderID,
	).Scan(&s.ID, &s.OrderID, &s.TrackingNo, &s.Carrier, &s.Status, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tracks, _ := r.listTracks(s.ID)
	s.Tracks = tracks
	return s, nil
}

func (r *ShipmentRepo) TrackByNo(no string) (*model.Shipment, error) {
	s := &model.Shipment{}
	err := r.db.QueryRow(
		`SELECT id, order_id, tracking_no, carrier, status, created_at FROM shipments WHERE tracking_no=?`, no,
	).Scan(&s.ID, &s.OrderID, &s.TrackingNo, &s.Carrier, &s.Status, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tracks, _ := r.listTracks(s.ID)
	s.Tracks = tracks
	return s, nil
}

func (r *ShipmentRepo) listTracks(shipmentID int64) ([]model.Track, error) {
	rows, err := r.db.Query(
		`SELECT id, shipment_id, description, location, occurred_at FROM shipment_tracks WHERE shipment_id=? ORDER BY occurred_at DESC`, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Track{}
	for rows.Next() {
		var t model.Track
		if err := rows.Scan(&t.ID, &t.ShipmentID, &t.Description, &t.Location, &t.OccurredAt); err == nil {
			out = append(out, t)
		}
	}
	return out, nil
}

// AdvanceStatus moves a shipment forward one step.
func (r *ShipmentRepo) AdvanceStatus(orderID int64) (*model.Shipment, error) {
	s, err := r.GetByOrder(orderID)
	if err != nil || s == nil {
		return nil, fmt.Errorf("shipment not found")
	}
	next := map[string]string{"shipped": "in_transit", "in_transit": "delivered"}
	nextDesc := map[string]string{
		"in_transit": "【菜鸟驿站】快件已到达配送站，快递员即将派送",
		"delivered":  "【天猫超市】您的订单已送达，感谢光临天猫，期待下次相见",
	}
	ns, ok := next[s.Status]
	if !ok {
		return s, nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE shipments SET status=? WHERE id=?`, ns, s.ID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(
		`INSERT INTO shipment_tracks (shipment_id, description, location, occurred_at) VALUES (?,?,?,?)`,
		s.ID, nextDesc[ns], "配送中", time.Now()); err != nil {
		return nil, err
	}
	if ns == "delivered" {
		if _, err := tx.Exec(`UPDATE orders SET status='completed' WHERE id=?`, orderID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByOrder(orderID)
}
