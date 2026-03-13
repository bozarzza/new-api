package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// MarketplaceChannelItem is the public-facing view of a marketplace channel (no key/sensitive info).
type MarketplaceChannelItem struct {
	Id             int     `json:"id"`
	Name           string  `json:"name"`
	Models         string  `json:"models"`
	PricePerKToken float64 `json:"price_per_k_token"`
	ChannelLabel   string  `json:"channel_label"`
	ResponseTime   int     `json:"response_time"`
	Status         int     `json:"status"`
	OwnerUserId    int     `json:"owner_user_id"`
	DailyTokenLimit int64  `json:"daily_token_limit"`
	DailyTokenUsed  int64  `json:"daily_token_used"`
	MaxConcurrent   int    `json:"max_concurrent"`
	AvgRating      float64 `json:"avg_rating"`
	RatingCount    int     `json:"rating_count"`
}

func channelToMarketplaceItem(ch *model.Channel) MarketplaceChannelItem {
	item := MarketplaceChannelItem{
		Id:              ch.Id,
		Name:            ch.Name,
		Models:          ch.Models,
		PricePerKToken:  ch.PricePerKToken,
		ChannelLabel:    ch.GetChannelLabel(),
		ResponseTime:    ch.ResponseTime,
		Status:          ch.Status,
		OwnerUserId:     ch.OwnerUserId,
		DailyTokenLimit: ch.DailyTokenLimit,
		DailyTokenUsed:  ch.DailyTokenUsed,
		MaxConcurrent:   ch.MaxConcurrent,
	}
	// Attach rating stats
	stats, err := model.GetChannelRatingStats(ch.Id)
	if err == nil {
		item.AvgRating = stats.AvgScore
		item.RatingCount = stats.Count
	}
	return item
}

// ListMarketplaceChannels returns public marketplace channels for buyers to browse.
// GET /api/marketplace/channels?model=xxx&label=xxx&min_price=0&max_price=100&p=1&page_size=20
func ListMarketplaceChannels(c *gin.Context) {
	modelFilter := c.Query("model")
	labelFilter := c.Query("label")
	minPriceStr := c.Query("min_price")
	maxPriceStr := c.Query("max_price")

	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	// Build query for marketplace channels only
	query := model.DB.Model(&model.Channel{}).
		Where("channel_mode = ?", 1).
		Where("status = ?", common.ChannelStatusEnabled).
		Omit("key")

	if modelFilter != "" {
		modelsCol := "`models`"
		if common.UsingPostgreSQL {
			modelsCol = `"models"`
		}
		query = query.Where(modelsCol+" LIKE ?", "%"+modelFilter+"%")
	}
	if labelFilter != "" {
		query = query.Where("channel_label = ?", labelFilter)
	}
	if minPriceStr != "" {
		if minPrice, err := strconv.ParseFloat(minPriceStr, 64); err == nil {
			query = query.Where("price_per_k_token >= ?", minPrice)
		}
	}
	if maxPriceStr != "" {
		if maxPrice, err := strconv.ParseFloat(maxPriceStr, 64); err == nil {
			query = query.Where("price_per_k_token <= ?", maxPrice)
		}
	}

	var total int64
	query.Count(&total)

	var channels []*model.Channel
	err := query.Order("price_per_k_token ASC, response_time ASC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&channels).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}

	items := make([]MarketplaceChannelItem, 0, len(channels))
	for _, ch := range channels {
		items = append(items, channelToMarketplaceItem(ch))
	}

	common.ApiSuccess(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ListSellerChannels returns marketplace channels owned by the current user.
// GET /api/marketplace/channels/self
func ListSellerChannels(c *gin.Context) {
	userId := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	query := model.DB.Model(&model.Channel{}).
		Where("owner_user_id = ? AND channel_mode = ?", userId, 1).
		Omit("key")

	var total int64
	query.Count(&total)

	var channels []*model.Channel
	err := query.Order("id desc").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&channels).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}

	items := make([]MarketplaceChannelItem, 0, len(channels))
	for _, ch := range channels {
		items = append(items, channelToMarketplaceItem(ch))
	}

	common.ApiSuccess(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AddSellerChannelRequest is the request body for seller channel submission.
type AddSellerChannelRequest struct {
	Name           string  `json:"name" binding:"required"`
	BaseURL        string  `json:"base_url" binding:"required"`
	Key            string  `json:"key" binding:"required"`
	Models         string  `json:"models" binding:"required"`
	PricePerKToken float64 `json:"price_per_k_token" binding:"required"`
	ChannelLabel   string  `json:"channel_label"` // "official" or "relay"
	DailyTokenLimit int64  `json:"daily_token_limit"`
	MaxConcurrent   int    `json:"max_concurrent"`
	Type           int     `json:"type"` // channel type (e.g., OpenAI compatible = 1)
}

// AddSellerChannel allows a user to submit their channel to the marketplace.
// POST /api/marketplace/channels
func AddSellerChannel(c *gin.Context) {
	userId := c.GetInt("id")
	var req AddSellerChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误: " + err.Error()})
		return
	}

	if req.PricePerKToken <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "价格必须大于0"})
		return
	}

	if req.Type == 0 {
		req.Type = 1 // default to OpenAI compatible
	}

	channel := &model.Channel{
		Name:            req.Name,
		Type:            req.Type,
		Key:             req.Key,
		BaseURL:         &req.BaseURL,
		Models:          req.Models,
		Status:          common.ChannelStatusEnabled,
		CreatedTime:     common.GetTimestamp(),
		OwnerUserId:     userId,
		ChannelMode:     1, // marketplace
		PricePerKToken:  req.PricePerKToken,
		DailyTokenLimit: req.DailyTokenLimit,
		MaxConcurrent:   req.MaxConcurrent,
		Group:           "default",
	}
	if req.ChannelLabel != "" {
		channel.ChannelLabel = &req.ChannelLabel
	}

	if err := channel.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "渠道已上架",
		"data":    gin.H{"id": channel.Id},
	})
}

// UpdateSellerChannel allows a seller to update their own marketplace channel.
// PUT /api/marketplace/channels/:id
func UpdateSellerChannel(c *gin.Context) {
	userId := c.GetInt("id")
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, fmt.Errorf("无效的渠道ID"))
		return
	}

	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// Verify ownership
	if channel.OwnerUserId != userId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权操作此渠道"})
		return
	}
	if !channel.IsMarketplaceChannel() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "此渠道不是市场渠道"})
		return
	}

	// Parse update fields
	var req struct {
		Name            *string  `json:"name"`
		Models          *string  `json:"models"`
		PricePerKToken  *float64 `json:"price_per_k_token"`
		ChannelLabel    *string  `json:"channel_label"`
		DailyTokenLimit *int64   `json:"daily_token_limit"`
		MaxConcurrent   *int     `json:"max_concurrent"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	if req.Name != nil {
		channel.Name = *req.Name
	}
	if req.Models != nil {
		channel.Models = *req.Models
	}
	if req.PricePerKToken != nil {
		if *req.PricePerKToken <= 0 {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "价格必须大于0"})
			return
		}
		channel.PricePerKToken = *req.PricePerKToken
	}
	if req.ChannelLabel != nil {
		channel.ChannelLabel = req.ChannelLabel
	}
	if req.DailyTokenLimit != nil {
		channel.DailyTokenLimit = *req.DailyTokenLimit
	}
	if req.MaxConcurrent != nil {
		channel.MaxConcurrent = *req.MaxConcurrent
	}

	if err := channel.Update(); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "更新成功"})
}

// DeleteSellerChannel allows a seller to remove their own marketplace channel.
// DELETE /api/marketplace/channels/:id
func DeleteSellerChannel(c *gin.Context) {
	userId := c.GetInt("id")
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, fmt.Errorf("无效的渠道ID"))
		return
	}

	channel, err := model.GetChannelById(channelId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if channel.OwnerUserId != userId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权操作此渠道"})
		return
	}

	if err := channel.Delete(); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "渠道已下架"})
}

// RateChannel allows a buyer to rate a marketplace channel.
// POST /api/marketplace/channels/:id/rate
func RateChannel(c *gin.Context) {
	userId := c.GetInt("id")
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, fmt.Errorf("无效的渠道ID"))
		return
	}

	var req struct {
		Score   int    `json:"score" binding:"required"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	// Check if already rated
	if model.HasUserRatedChannel(userId, channelId) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "您已评价过此渠道"})
		return
	}

	rating := &model.Rating{
		UserId:    userId,
		ChannelId: channelId,
		Score:     req.Score,
		Comment:   req.Comment,
	}
	if err := model.CreateRating(rating); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "评价成功"})
}

// GetChannelRatingsAPI returns paginated ratings for a channel.
// GET /api/marketplace/channels/:id/ratings
func GetChannelRatingsAPI(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, fmt.Errorf("无效的渠道ID"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	ratings, total, err := model.GetChannelRatings(channelId, (page-1)*pageSize, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	stats, _ := model.GetChannelRatingStats(channelId)

	common.ApiSuccess(c, gin.H{
		"items":     ratings,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"avg_score": stats.AvgScore,
	})
}
