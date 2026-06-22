package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/tmall-clone/backend/internal/handler"
)

func New(h *handler.Handler, allowedOrigins string) *gin.Engine {
	r := gin.Default()
	r.Use(corsMiddleware(allowedOrigins))

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	// Serve uploaded images from the local data/images directory.
	r.Static("/images", "data/images")

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", h.Login)
		api.POST("/auth/register", h.Register)

		api.GET("/categories", h.ListCategories)
		api.GET("/brands", h.ListBrands)
		api.GET("/brands/:id", h.GetBrand)
		api.GET("/products", h.ListProducts)
		api.GET("/products/:id", h.GetProduct)
		api.GET("/products/:id/reviews", h.ListReviews)
		api.GET("/products/:id/skus", h.ListSKUs)

		api.GET("/shipments/track", h.TrackByNo)

		auth := api.Group("/")
		auth.Use(authMiddleware())
		{
			auth.GET("/auth/profile", h.Profile)

			auth.GET("/cart", h.ListCart)
			auth.POST("/cart", h.AddCart)
			auth.PUT("/cart/:id", h.UpdateCart)
			auth.DELETE("/cart/:id", h.DeleteCart)

			auth.GET("/orders", h.ListOrders)
			auth.POST("/orders", h.CreateOrder)
			auth.POST("/orders/:id/pay", h.PayOrder)

			// Payment flow (sandbox)
			auth.POST("/payments", h.CreatePayment)
			auth.POST("/payments/:id/confirm", h.ConfirmPayment)
			auth.GET("/payments/order/:id", h.GetPayment)

			// Shipment flow
			auth.POST("/orders/:id/ship", h.ShipOrder)
			auth.GET("/orders/:id/track", h.TrackOrder)
			auth.POST("/orders/:id/ship/advance", h.AdvanceShipment)

			auth.POST("/reviews", h.CreateReview)

			// Image upload (admin product form + review photos)
			auth.POST("/upload", h.UploadImage)

			auth.POST("/admin/products", h.AdminCreateProduct)
			auth.PUT("/admin/products/:id", h.AdminUpdateProduct)
			auth.DELETE("/admin/products/:id", h.AdminDeleteProduct)
			auth.POST("/admin/products/:id/skus", h.CreateSKU)
		}
	}
	return r
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
			return
		}
		c.Next()
	}
}

func corsMiddleware(allowed string) gin.HandlerFunc {
	allowAll := strings.TrimSpace(allowed) == "*" || allowed == ""
	origins := map[string]bool{}
	for _, o := range strings.Split(allowed, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins[o] = true
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		switch {
		case allowAll:
			c.Header("Access-Control-Allow-Origin", "*")
		case origin != "" && origins[origin]:
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
