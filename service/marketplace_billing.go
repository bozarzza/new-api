package service

import (
	"fmt"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

// MarketplaceCommissionRate is the platform commission rate (e.g., 0.10 = 10%).
// TODO: make this configurable via admin settings.
const MarketplaceCommissionRate = 0.10

// SettleMarketplaceBilling handles credit settlement for marketplace channels.
// Called after standard billing completes. Credits the seller (minus commission)
// and updates daily token counter using quota as proxy for usage.
func SettleMarketplaceBilling(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, actualQuota int) {
	if actualQuota <= 0 {
		return
	}

	channelId := relayInfo.ChannelId
	if channelId == 0 {
		return
	}

	// Load channel to check if it's a marketplace channel
	channel, err := model.CacheGetChannel(channelId)
	if err != nil {
		// Fallback to DB
		channel, err = model.GetChannelById(channelId, false)
		if err != nil {
			return
		}
	}

	if !channel.IsMarketplaceChannel() || channel.OwnerUserId == 0 {
		return
	}

	// Calculate seller earnings (quota minus commission)
	commission := int64(float64(actualQuota) * MarketplaceCommissionRate)
	sellerEarning := int64(actualQuota) - commission

	if sellerEarning <= 0 {
		return
	}

	// Credit the seller
	if err := model.IncreaseCreditBalance(channel.OwnerUserId, sellerEarning); err != nil {
		logger.LogError(ctx, fmt.Sprintf("marketplace: failed to credit seller %d for channel %d: %s",
			channel.OwnerUserId, channelId, err.Error()))
		return
	}

	// Record seller earning transaction
	if err := model.RecordCreditTransaction(
		channel.OwnerUserId,
		model.CreditTypeEarn,
		sellerEarning,
		channelId,
		fmt.Sprintf("渠道 #%d 使用收益 (扣除 %.0f%% 平台佣金)", channelId, MarketplaceCommissionRate*100),
	); err != nil {
		logger.LogError(ctx, fmt.Sprintf("marketplace: failed to record seller transaction: %s", err.Error()))
	}

	// Record platform commission transaction (fee)
	if commission > 0 {
		if err := model.RecordCreditTransaction(
			channel.OwnerUserId,
			model.CreditTypeFee,
			-commission,
			channelId,
			fmt.Sprintf("渠道 #%d 平台佣金 (%.0f%%)", channelId, MarketplaceCommissionRate*100),
		); err != nil {
			logger.LogError(ctx, fmt.Sprintf("marketplace: failed to record fee transaction: %s", err.Error()))
		}
	}

	// Update daily token counter (using quota as proxy for usage)
	model.UpdateChannelDailyTokenUsed(channelId, int64(actualQuota))
}
