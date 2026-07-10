package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GenerateGiftCard: POST /gift-cards/generate — create a demo gift card.
// In a real system this would be an admin/minting operation; here it is open
// (under auth) so the gift-card demo page can mint a card for testing.
func (h *Handler) GenerateGiftCard(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	if h.GiftCard == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "礼品卡功能未开启"})
		return
	}
	var req struct {
		Amount float64 `json:"amount"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Amount <= 0 {
		req.Amount = 100
	}
	gc, err := h.GiftCard.Generate(req.Amount, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gc, "message": "礼品卡生成成功"})
}

// RedeemGiftCard: POST /gift-cards/redeem — bind a gift card to the current user.
func (h *Handler) RedeemGiftCard(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	if h.GiftCard == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "礼品卡功能未开启"})
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入礼品卡卡号"})
		return
	}
	gc, err := h.GiftCard.Redeem(req.Code, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	list, _ := h.GiftCard.ListByUser(uid)
	c.JSON(http.StatusOK, gin.H{"data": gc, "cards": list, "message": "兑换成功，礼品卡已绑定"})
}

// ListGiftCards: GET /gift-cards — the current user's redeemed gift cards.
func (h *Handler) ListGiftCards(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	if h.GiftCard == nil {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}
	list, err := h.GiftCard.ListByUser(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	total := 0.0
	for _, gc := range list {
		if gc.Status == "used" {
			total += gc.Amount
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "total": strconv.FormatFloat(total, 'f', 2, 64)})
}
