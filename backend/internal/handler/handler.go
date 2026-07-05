package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/tmall-clone/backend/internal/model"
	"github.com/CodingFervor/tmall-clone/backend/internal/repository"
)

// Handler bundles all repositories and the JWT secret.
type Handler struct {
	User         *repository.UserRepo
	Brand        *repository.BrandRepo
	Cat          *repository.CategoryRepo
	Product      *repository.ProductRepo
	Cart         *repository.CartRepo
	Order        *repository.OrderRepo
	Review       *repository.ReviewRepo
	SKU          *repository.SKURepo
	Payment      *repository.PaymentRepo
	Shipment     *repository.ShipmentRepo
	Refund       *repository.RefundRepo
	Coupon       *repository.CouponRepo
	Address      *repository.AddressRepo
	Favorite     *repository.FavoriteRepo
	History      *repository.HistoryRepo
	CheckIn      *repository.CheckInRepo
	Shop         *repository.PointShopRepo
	Seckill      *repository.SeckillRepo
	GroupBuy     *repository.GroupBuyRepo
	Presale      *repository.PresaleRepo
	PriceHistory *repository.PriceHistoryRepo
	ShopRating   *repository.ShopRatingRepo
	Bundle       *repository.BundleRepo
	Restock      *repository.RestockRepo
	QA           *repository.QARepo
	Invoice      *repository.InvoiceRepo
	Tiered       *repository.TieredDiscountRepo
	jwtKey       []byte
}

func New(jwtSecret string, u *repository.UserRepo, b *repository.BrandRepo, c *repository.CategoryRepo, p *repository.ProductRepo,
	ca *repository.CartRepo, o *repository.OrderRepo, r *repository.ReviewRepo) *Handler {
	return &Handler{User: u, Brand: b, Cat: c, Product: p, Cart: ca, Order: o, Review: r, jwtKey: []byte(jwtSecret)}
}

// SetUserExtra attaches the address + favorite (wishlist) repos.
func (h *Handler) SetUserExtra(addr *repository.AddressRepo, fav *repository.FavoriteRepo) {
	h.Address = addr
	h.Favorite = fav
}

// SetHistory attaches the browse-history + check-in + points-mall repos.
func (h *Handler) SetHistory(hist *repository.HistoryRepo, ci *repository.CheckInRepo, shop *repository.PointShopRepo) {
	h.History = hist
	h.CheckIn = ci
	h.Shop = shop
}

// SetSeckill attaches the flash-sale repo.
func (h *Handler) SetSeckill(s *repository.SeckillRepo) {
	h.Seckill = s
}

// SetGroupBuy attaches the group-buy repo.
func (h *Handler) SetGroupBuy(g *repository.GroupBuyRepo) {
	h.GroupBuy = g
}

// SetPresale attaches the presale repo.
func (h *Handler) SetPresale(p *repository.PresaleRepo) {
	h.Presale = p
}

// SetPriceHistory attaches the price-history repo.
func (h *Handler) SetPriceHistory(ph *repository.PriceHistoryRepo) {
	h.PriceHistory = ph
}

// SetShopRating attaches the shop-rating repo.
func (h *Handler) SetShopRating(sr *repository.ShopRatingRepo) {
	h.ShopRating = sr
}

// SetBundle attaches the bundle + restock repos.
func (h *Handler) SetBundle(b *repository.BundleRepo, r *repository.RestockRepo) {
	h.Bundle = b
	h.Restock = r
}

// SetQA attaches the product-Q&A repo.
func (h *Handler) SetQA(qa *repository.QARepo) {
	h.QA = qa
}

// SetInvoice attaches the order-invoice repo.
func (h *Handler) SetInvoice(inv *repository.InvoiceRepo) {
	h.Invoice = inv
}

// SetTiered attaches the tiered-discount (阶梯满减) repo.
func (h *Handler) SetTiered(td *repository.TieredDiscountRepo) {
	h.Tiered = td
}

// ---- JWT (HS256, hand-rolled) ----

func (h *Handler) signToken(userID int64, username string) string {
	header := `{"alg":"HS256","typ":"JWT"}`
	payload := `{"user_id":` + strconv.FormatInt(userID, 10) + `,"username":"` + username + `","exp":` + strconv.FormatInt(time.Now().Add(72*time.Hour).Unix(), 10) + `}`
	return encodeSeg(header) + "." + encodeSeg(payload) + "." + h.signature(header, payload)
}

func (h *Handler) parseToken(token string) (int64, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, false
	}
	if h.signature(decodeSeg(parts[0]), decodeSeg(parts[1])) != parts[2] {
		return 0, false
	}
	payload := decodeSeg(parts[1])
	uid := extractInt(payload, "user_id")
	exp := extractInt(payload, "exp")
	if exp > 0 && time.Now().Unix() > exp {
		return 0, false
	}
	return uid, true
}

func (h *Handler) signature(header, payload string) string {
	sum := sha256.Sum256([]byte(header + "." + payload + "." + string(h.jwtKey)))
	return hex.EncodeToString(sum[:])
}

func (h *Handler) currentUserID(c *gin.Context, optional ...bool) (int64, bool) {
	auth := c.GetHeader("Authorization")
	tok := strings.TrimPrefix(auth, "Bearer ")
	uid, ok := h.parseToken(tok)
	if !ok {
		if len(optional) > 0 && optional[0] {
			return 0, true
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return 0, false
	}
	return uid, true
}

func hashPassword(plain string) string {
	sum := sha256.Sum256([]byte(plain + "tmall-salt"))
	return hex.EncodeToString(sum[:])
}

// ---- Auth ----

func (h *Handler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码必填"})
		return
	}
	u, err := h.User.FindByUsername(req.Username)
	if err != nil || u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	if u.Password != req.Password && u.Password != hashPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": h.signToken(u.ID, u.Username), "user": u})
}

func (h *Handler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	if h.User.Exists(req.Username) {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}
	u := &model.User{Username: req.Username, Password: hashPassword(req.Password), Nickname: req.Nickname}
	if u.Nickname == "" {
		u.Nickname = req.Username
	}
	u.Avatar = "https://api.dicebear.com/7.x/avataaars/svg?seed=" + req.Username
	if err := h.User.Create(u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "注册失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": h.signToken(u.ID, u.Username), "user": u})
}

func (h *Handler) Profile(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	u, err := h.User.Get(uid)
	if err != nil || u == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	count, _ := h.Cart.Count(uid)
	c.JSON(http.StatusOK, gin.H{"user": u, "cart_count": count})
}
