package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetCreditBalance returns the credit balance for the current user.
// GET /api/credit/balance
func GetCreditBalance(c *gin.Context) {
	userId := c.GetInt("id")
	account, err := model.GetOrCreateCreditAccount(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"balance":      account.Balance,
		"total_earned": account.TotalEarned,
		"total_spent":  account.TotalSpent,
	})
}

// GetCreditTransactions returns paginated credit transactions for the current user.
// GET /api/credit/transactions?p=1&page_size=20
func GetCreditTransactions(c *gin.Context) {
	userId := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	transactions, total, err := model.GetCreditTransactions(userId, (page-1)*pageSize, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"items":     transactions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// RequestWithdrawal creates a withdrawal request (pending admin approval).
// POST /api/credit/withdraw
func RequestWithdrawal(c *gin.Context) {
	userId := c.GetInt("id")

	var req struct {
		Amount      int64  `json:"amount" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误: " + err.Error()})
		return
	}

	if req.Amount <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "提现金额必须大于0"})
		return
	}

	// Check balance
	balance, err := model.GetCreditBalance(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if balance < req.Amount {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "积分余额不足"})
		return
	}

	// Deduct balance and record transaction
	if err := model.DecreaseCreditBalance(userId, req.Amount); err != nil {
		common.ApiError(c, err)
		return
	}

	description := "提现申请"
	if req.Description != "" {
		description = req.Description
	}
	if err := model.RecordCreditTransaction(userId, model.CreditTypeWithdraw, -req.Amount, 0, description); err != nil {
		// Balance already deducted, log the error but don't fail
		common.SysError("failed to record withdrawal transaction: " + err.Error())
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "提现申请已提交，请等待管理员处理",
	})
}
