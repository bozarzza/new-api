package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
)

// Rating stores buyer feedback for a seller's channel.
type Rating struct {
	Id        int    `json:"id"`
	UserId    int    `json:"user_id" gorm:"index;not null"`    // buyer who rated
	ChannelId int    `json:"channel_id" gorm:"index;not null"` // seller's channel
	Score     int    `json:"score" gorm:"not null"`            // 1-5
	Comment   string `json:"comment" gorm:"type:varchar(255)"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index"`
}

// CreateRating inserts a new rating after validation.
func CreateRating(rating *Rating) error {
	if rating.Score < 1 || rating.Score > 5 {
		return fmt.Errorf("score must be between 1 and 5, got %d", rating.Score)
	}
	if rating.UserId == 0 {
		return errors.New("user_id is required")
	}
	if rating.ChannelId == 0 {
		return errors.New("channel_id is required")
	}
	rating.CreatedAt = common.GetTimestamp()
	return DB.Create(rating).Error
}

// ChannelRatingStats holds aggregated rating data.
type ChannelRatingStats struct {
	AvgScore float64 `json:"avg_score"`
	Count    int     `json:"count"`
}

// GetChannelRatingStats returns the average score and total count for a channel.
func GetChannelRatingStats(channelId int) (ChannelRatingStats, error) {
	var stats ChannelRatingStats
	err := DB.Model(&Rating{}).
		Select("COALESCE(AVG(score), 0) as avg_score, COUNT(*) as count").
		Where("channel_id = ?", channelId).
		Scan(&stats).Error
	return stats, err
}

// GetChannelRatings returns paginated ratings for a channel.
func GetChannelRatings(channelId int, startIdx int, num int) ([]Rating, int64, error) {
	var ratings []Rating
	var total int64

	tx := DB.Model(&Rating{}).Where("channel_id = ?", channelId)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Order("id desc").Limit(num).Offset(startIdx).Find(&ratings).Error; err != nil {
		return nil, 0, err
	}
	return ratings, total, nil
}

// HasUserRatedChannel checks if a user has already rated a specific channel.
func HasUserRatedChannel(userId int, channelId int) bool {
	var count int64
	DB.Model(&Rating{}).Where("user_id = ? AND channel_id = ?", userId, channelId).Count(&count)
	return count > 0
}
