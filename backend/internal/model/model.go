package model

import "time"

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
}

// Brand is a flagship store brand (天猫旗舰店 concept).
type Brand struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Logo        string `json:"logo"`
	Description string `json:"description"`
	Followers   int    `json:"followers"`
	SortOrder   int    `json:"sort_order"`
}

type Category struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
}

type Product struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Subtitle      string    `json:"subtitle"`
	Price         float64   `json:"price"`
	OriginalPrice float64   `json:"original_price"`
	Image         string    `json:"image"`
	Images        string    `json:"images"`
	Category      string    `json:"category"`
	CategoryID    int64     `json:"category_id"`
	BrandID       int64     `json:"brand_id"`
	BrandName     string    `json:"brand_name"`
	Shop          string    `json:"shop"`
	Stock         int       `json:"stock"`
	Sales         int       `json:"sales"`
	Description   string    `json:"description"`
	Tags          string    `json:"tags"`
	IsGenuine     int       `json:"is_genuine"`
	CreatedAt     time.Time `json:"created_at"`
}

type CartItem struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	ProductID   int64     `json:"product_id"`
	Quantity    int       `json:"quantity"`
	Selected    int       `json:"selected"`
	ProductName string    `json:"product_name"`
	ProductImg  string    `json:"product_image"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
}

type Order struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	OrderNo   string    `json:"order_no"`
	Total     float64   `json:"total"`
	Status    string    `json:"status"`
	ItemsJSON string    `json:"items_json"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
}

type Review struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Rating    int       `json:"rating"`
	Content   string    `json:"content"`
	Images    string    `json:"images"`
	CreatedAt time.Time `json:"created_at"`
}

// SKU is a specific spec combination of a product.
type SKU struct {
	ID        int64   `json:"id"`
	ProductID int64   `json:"product_id"`
	Spec      string  `json:"spec"`
	SpecText  string  `json:"spec_text"`
	Price     float64 `json:"price"`
	Stock     int     `json:"stock"`
	SKUCode   string  `json:"sku_code"`
}

// Shipment is a shipped order's logistics envelope.
type Shipment struct {
	ID         int64     `json:"id"`
	OrderID    int64     `json:"order_id"`
	TrackingNo string    `json:"tracking_no"`
	Carrier    string    `json:"carrier"`
	Status     string    `json:"status"`
	Tracks     []Track   `json:"tracks"`
	CreatedAt  time.Time `json:"created_at"`
}

// Track is one logistics event.
type Track struct {
	ID          int64     `json:"id"`
	ShipmentID  int64     `json:"shipment_id"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// Payment records a payment attempt for an order.
type Payment struct {
	ID            int64     `json:"id"`
	OrderID       int64     `json:"order_id"`
	UserID        int64     `json:"user_id"`
	Amount        float64   `json:"amount"`
	Method        string    `json:"method"`
	TransactionNo string    `json:"transaction_no"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// ---- Request DTOs ----

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname"`
}

type AddCartRequest struct {
	ProductID int64 `json:"product_id" binding:"required"`
	Quantity  int   `json:"quantity"`
}

type UpdateCartRequest struct {
	Quantity int `json:"quantity"`
	Selected int `json:"selected"`
}

type CreateOrderRequest struct {
	Items   []OrderItemInput `json:"items" binding:"required"`
	Address string           `json:"address"`
}
type OrderItemInput struct {
	ProductID int64 `json:"product_id" binding:"required"`
	Quantity  int   `json:"quantity" binding:"required"`
}

type CreateReviewRequest struct {
	ProductID int64  `json:"product_id" binding:"required"`
	Rating    int    `json:"rating"`
	Content   string `json:"content"`
}

type ProductInput struct {
	Name          string  `json:"name" binding:"required"`
	Subtitle      string  `json:"subtitle"`
	Price         float64 `json:"price" binding:"required"`
	OriginalPrice float64 `json:"original_price"`
	Image         string  `json:"image"`
	Images        string  `json:"images"`
	Category      string  `json:"category"`
	CategoryID    int64   `json:"category_id"`
	BrandID       int64   `json:"brand_id"`
	BrandName     string  `json:"brand_name"`
	Shop          string  `json:"shop"`
	Stock         int     `json:"stock"`
	Sales         int     `json:"sales"`
	Description   string  `json:"description"`
	Tags          string  `json:"tags"`
	IsGenuine     int     `json:"is_genuine"`
}
