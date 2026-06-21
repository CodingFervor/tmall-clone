package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/CodingFervor/tmall-clone/backend/internal/model"
)

// ===================== User =====================

type UserRepo struct{ db *sql.DB }

func NewUserRepo(db *sql.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(u *model.User) error {
	res, err := r.db.Exec(
		`INSERT INTO users (username, password, nickname, avatar) VALUES (?,?,?,?)`,
		u.Username, u.Password, defaultStr(u.Nickname, u.Username), u.Avatar)
	if err != nil {
		return err
	}
	u.ID, _ = res.LastInsertId()
	return nil
}

func (r *UserRepo) FindByUsername(username string) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, username, password, nickname, avatar, created_at FROM users WHERE username=?`, username,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Nickname, &u.Avatar, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *UserRepo) Get(id int64) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, username, password, nickname, avatar, created_at FROM users WHERE id=?`, id,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Nickname, &u.Avatar, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *UserRepo) Exists(username string) bool {
	var n int
	_ = r.db.QueryRow(`SELECT 1 FROM users WHERE username=? LIMIT 1`, username).Scan(&n)
	return n == 1
}

// ===================== Brand =====================

type BrandRepo struct{ db *sql.DB }

func NewBrandRepo(db *sql.DB) *BrandRepo { return &BrandRepo{db: db} }

func (r *BrandRepo) All() ([]model.Brand, error) {
	rows, err := r.db.Query(`SELECT id, name, logo, description, followers, sort_order FROM brands ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Brand{}
	for rows.Next() {
		var b model.Brand
		if err := rows.Scan(&b.ID, &b.Name, &b.Logo, &b.Description, &b.Followers, &b.SortOrder); err == nil {
			out = append(out, b)
		}
	}
	return out, nil
}

func (r *BrandRepo) Get(id int64) (*model.Brand, error) {
	b := &model.Brand{}
	err := r.db.QueryRow(
		`SELECT id, name, logo, description, followers, sort_order FROM brands WHERE id=?`, id,
	).Scan(&b.ID, &b.Name, &b.Logo, &b.Description, &b.Followers, &b.SortOrder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return b, err
}

func (r *BrandRepo) Count() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM brands`).Scan(&n)
	return n, err
}

// ===================== Category =====================

type CategoryRepo struct{ db *sql.DB }

func NewCategoryRepo(db *sql.DB) *CategoryRepo { return &CategoryRepo{db: db} }

func (r *CategoryRepo) All() ([]model.Category, error) {
	rows, err := r.db.Query(`SELECT id, name, icon, sort_order FROM categories ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Category{}
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder); err == nil {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *CategoryRepo) Count() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&n)
	return n, err
}

// ===================== Product =====================

type ProductRepo struct{ db *sql.DB }

func NewProductRepo(db *sql.DB) *ProductRepo { return &ProductRepo{db: db} }

func (r *ProductRepo) List(page, pageSize int, categoryID, brandID int64, keyword string) ([]model.Product, int, error) {
	where := ""
	args := []any{}
	if categoryID > 0 {
		where = "WHERE category_id=?"
		args = append(args, categoryID)
	}
	if brandID > 0 {
		if where == "" {
			where = "WHERE brand_id=?"
		} else {
			where += " AND brand_id=?"
		}
		args = append(args, brandID)
	}
	if keyword != "" {
		if where == "" {
			where = "WHERE name LIKE ?"
		} else {
			where += " AND name LIKE ?"
		}
		args = append(args, "%"+keyword+"%")
	}
	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	err := r.db.QueryRow("SELECT COUNT(*) FROM products "+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := "SELECT id,name,subtitle,price,original_price,image,images,category,category_id,brand_id,brand_name,shop,stock,sales,description,tags,is_genuine,created_at FROM products " +
		where + " ORDER BY sales DESC, id DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.Product{}
	for rows.Next() {
		var p model.Product
		if err := scanProduct(rows, &p); err == nil {
			out = append(out, p)
		}
	}
	return out, total, nil
}

func (r *ProductRepo) ListByBrand(brandID int64, limit int) ([]model.Product, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(
		`SELECT id,name,subtitle,price,original_price,image,images,category,category_id,brand_id,brand_name,shop,stock,sales,description,tags,is_genuine,created_at
		 FROM products WHERE brand_id=? ORDER BY sales DESC LIMIT ?`, brandID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Product{}
	for rows.Next() {
		var p model.Product
		if err := scanProduct(rows, &p); err == nil {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *ProductRepo) Get(id int64) (*model.Product, error) {
	p := &model.Product{}
	row := r.db.QueryRow(
		`SELECT id,name,subtitle,price,original_price,image,images,category,category_id,brand_id,brand_name,shop,stock,sales,description,tags,is_genuine,created_at
		 FROM products WHERE id=?`, id)
	if err := scanProductRow(row, p); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *ProductRepo) Create(p *model.ProductInput) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO products (name,subtitle,price,original_price,image,images,category,category_id,brand_id,brand_name,shop,stock,sales,description,tags,is_genuine)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.Name, p.Subtitle, p.Price, p.OriginalPrice, p.Image, p.Images, p.Category, p.CategoryID,
		p.BrandID, p.BrandName, p.Shop, defaultInt(p.Stock, 999), p.Sales, p.Description, p.Tags, p.IsGenuine)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ProductRepo) Update(id int64, p *model.ProductInput) error {
	_, err := r.db.Exec(
		`UPDATE products SET name=?,subtitle=?,price=?,original_price=?,image=?,images=?,category=?,category_id=?,brand_id=?,brand_name=?,shop=?,stock=?,sales=?,description=?,tags=?,is_genuine=? WHERE id=?`,
		p.Name, p.Subtitle, p.Price, p.OriginalPrice, p.Image, p.Images, p.Category, p.CategoryID,
		p.BrandID, p.BrandName, p.Shop, p.Stock, p.Sales, p.Description, p.Tags, p.IsGenuine, id)
	return err
}

func (r *ProductRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM products WHERE id=?`, id)
	return err
}

func scanProduct(rows *sql.Rows, p *model.Product) error {
	return rows.Scan(&p.ID, &p.Name, &p.Subtitle, &p.Price, &p.OriginalPrice, &p.Image, &p.Images,
		&p.Category, &p.CategoryID, &p.BrandID, &p.BrandName, &p.Shop, &p.Stock, &p.Sales, &p.Description, &p.Tags, &p.IsGenuine, &p.CreatedAt)
}

func scanProductRow(row *sql.Row, p *model.Product) error {
	return row.Scan(&p.ID, &p.Name, &p.Subtitle, &p.Price, &p.OriginalPrice, &p.Image, &p.Images,
		&p.Category, &p.CategoryID, &p.BrandID, &p.BrandName, &p.Shop, &p.Stock, &p.Sales, &p.Description, &p.Tags, &p.IsGenuine, &p.CreatedAt)
}

// ===================== Cart =====================

type CartRepo struct{ db *sql.DB }

func NewCartRepo(db *sql.DB) *CartRepo { return &CartRepo{db: db} }

func (r *CartRepo) List(userID int64) ([]model.CartItem, error) {
	rows, err := r.db.Query(
		`SELECT c.id, c.user_id, c.product_id, c.quantity, c.selected, c.created_at,
		        p.name, p.image, p.price, p.stock
		 FROM cart_items c JOIN products p ON p.id = c.product_id
		 WHERE c.user_id=? ORDER BY c.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.CartItem{}
	for rows.Next() {
		var c model.CartItem
		if err := rows.Scan(&c.ID, &c.UserID, &c.ProductID, &c.Quantity, &c.Selected, &c.CreatedAt,
			&c.ProductName, &c.ProductImg, &c.Price, &c.Stock); err == nil {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *CartRepo) Add(userID, productID int64, qty int) error {
	if qty < 1 {
		qty = 1
	}
	var existingID int64
	err := r.db.QueryRow(`SELECT id FROM cart_items WHERE user_id=? AND product_id=?`, userID, productID).Scan(&existingID)
	if err == nil {
		_, err = r.db.Exec(`UPDATE cart_items SET quantity = quantity + ? WHERE id=?`, qty, existingID)
		return err
	}
	if err != sql.ErrNoRows {
		return err
	}
	_, err = r.db.Exec(`INSERT INTO cart_items (user_id, product_id, quantity) VALUES (?,?,?)`, userID, productID, qty)
	return err
}

func (r *CartRepo) Update(id, userID int64, qty, selected int) error {
	if qty < 0 {
		qty = 0
	}
	if qty == 0 {
		_, err := r.db.Exec(`DELETE FROM cart_items WHERE id=? AND user_id=?`, id, userID)
		return err
	}
	_, err := r.db.Exec(`UPDATE cart_items SET quantity=?, selected=? WHERE id=? AND user_id=?`, qty, selected, id, userID)
	return err
}

func (r *CartRepo) Delete(id, userID int64) error {
	_, err := r.db.Exec(`DELETE FROM cart_items WHERE id=? AND user_id=?`, id, userID)
	return err
}

func (r *CartRepo) Count(userID int64) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COALESCE(SUM(quantity),0) FROM cart_items WHERE user_id=?`, userID).Scan(&n)
	return n, err
}

func (r *CartRepo) Clear(userID int64) error {
	_, err := r.db.Exec(`DELETE FROM cart_items WHERE user_id=?`, userID)
	return err
}

// ===================== Order =====================

type OrderRepo struct{ db *sql.DB }

func NewOrderRepo(db *sql.DB) *OrderRepo { return &OrderRepo{db: db} }

func (r *OrderRepo) Create(o *model.Order) error {
	o.OrderNo = fmt.Sprintf("TM%d%d", time.Now().Unix(), o.UserID)
	res, err := r.db.Exec(
		`INSERT INTO orders (user_id, order_no, total, status, items_json, address) VALUES (?,?,?,?,?,?)`,
		o.UserID, o.OrderNo, o.Total, defaultStr(o.Status, "pending"), o.ItemsJSON, o.Address)
	if err != nil {
		return err
	}
	o.ID, _ = res.LastInsertId()
	return nil
}

func (r *OrderRepo) ListByUser(userID int64) ([]model.Order, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, order_no, total, status, items_json, address, created_at
		 FROM orders WHERE user_id=? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Order{}
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Total, &o.Status, &o.ItemsJSON, &o.Address, &o.CreatedAt); err == nil {
			out = append(out, o)
		}
	}
	return out, nil
}

func (r *OrderRepo) UpdateStatus(id, userID int64, status string) error {
	_, err := r.db.Exec(`UPDATE orders SET status=? WHERE id=? AND user_id=?`, status, id, userID)
	return err
}

// ===================== Review =====================

type ReviewRepo struct{ db *sql.DB }

func NewReviewRepo(db *sql.DB) *ReviewRepo { return &ReviewRepo{db: db} }

func (r *ReviewRepo) ListByProduct(productID int64) ([]model.Review, error) {
	rows, err := r.db.Query(
		`SELECT id, product_id, user_id, username, rating, content, images, created_at
		 FROM reviews WHERE product_id=? ORDER BY id DESC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Review{}
	for rows.Next() {
		var rv model.Review
		if err := rows.Scan(&rv.ID, &rv.ProductID, &rv.UserID, &rv.Username, &rv.Rating, &rv.Content, &rv.Images, &rv.CreatedAt); err == nil {
			out = append(out, rv)
		}
	}
	return out, nil
}

func (r *ReviewRepo) Create(rv *model.Review) error {
	if rv.Rating < 1 || rv.Rating > 5 {
		rv.Rating = 5
	}
	res, err := r.db.Exec(
		`INSERT INTO reviews (product_id, user_id, username, rating, content, images) VALUES (?,?,?,?,?,?)`,
		rv.ProductID, rv.UserID, rv.Username, rv.Rating, rv.Content, rv.Images)
	if err != nil {
		return err
	}
	rv.ID, _ = res.LastInsertId()
	return nil
}

// ===================== helpers =====================

func defaultStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func defaultInt(n, d int) int {
	if n == 0 {
		return d
	}
	return n
}
