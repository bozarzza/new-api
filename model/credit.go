package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// Credit transaction types
const (
	CreditTypeEarn     = 1 // seller earns from buyer usage
	CreditTypeSpend    = 2 // buyer spends on API calls
	CreditTypeWithdraw = 3 // seller withdraws to real money
	CreditTypeDeposit  = 4 // buyer deposits / converts from quota
	CreditTypeRefund   = 5 // refund
	CreditTypeFee      = 6 // platform commission deduction
)

// CreditAccount tracks credit balance for each user.
type CreditAccount struct {
	Id          int   `json:"id"`
	UserId      int   `json:"user_id" gorm:"uniqueIndex;not null"`
	Balance     int64 `json:"balance" gorm:"bigint;default:0"`
	TotalEarned int64 `json:"total_earned" gorm:"bigint;default:0"`
	TotalSpent  int64 `json:"total_spent" gorm:"bigint;default:0"`
	UpdatedAt   int64 `json:"updated_at" gorm:"bigint"`
}

// CreditTransaction records every credit movement for audit.
type CreditTransaction struct {
	Id          int    `json:"id"`
	UserId      int    `json:"user_id" gorm:"index;not null"`
	Type        int    `json:"type"`                                  // CreditType*
	Amount      int64  `json:"amount" gorm:"bigint;not null"`         // positive = credit in, negative = credit out
	Balance     int64  `json:"balance" gorm:"bigint"`                 // balance after this transaction
	RelatedId   int    `json:"related_id" gorm:"index;default:0"`    // related channel_id or log_id
	Description string `json:"description" gorm:"type:varchar(255)"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;index"`
}

// GetOrCreateCreditAccount returns the credit account for a user, creating one if needed.
func GetOrCreateCreditAccount(userId int) (*CreditAccount, error) {
	account := &CreditAccount{}
	err := DB.Where("user_id = ?", userId).First(account).Error
	if err == nil {
		return account, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	// Create new account
	account = &CreditAccount{
		UserId:    userId,
		Balance:   0,
		UpdatedAt: common.GetTimestamp(),
	}
	if err := DB.Create(account).Error; err != nil {
		// Handle race condition: another goroutine may have created it
		var existing CreditAccount
		if err2 := DB.Where("user_id = ?", userId).First(&existing).Error; err2 == nil {
			return &existing, nil
		}
		return nil, err
	}
	return account, nil
}

// GetCreditBalance returns the credit balance for a user.
func GetCreditBalance(userId int) (int64, error) {
	account, err := GetOrCreateCreditAccount(userId)
	if err != nil {
		return 0, err
	}
	return account.Balance, nil
}

// IncreaseCreditBalance atomically increases a user's credit balance.
func IncreaseCreditBalance(userId int, amount int64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	// Ensure account exists
	if _, err := GetOrCreateCreditAccount(userId); err != nil {
		return err
	}
	result := DB.Model(&CreditAccount{}).
		Where("user_id = ?", userId).
		Updates(map[string]interface{}{
			"balance":      gorm.Expr("balance + ?", amount),
			"total_earned": gorm.Expr("total_earned + ?", amount),
			"updated_at":   common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("credit account not found")
	}
	return nil
}

// DecreaseCreditBalance atomically decreases a user's credit balance.
// Returns error if balance is insufficient.
func DecreaseCreditBalance(userId int, amount int64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	// Use a transaction with row lock to prevent race conditions
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var account CreditAccount
	// Lock the row for update
	err := tx.Set("gorm:query_option", "FOR UPDATE").
		Where("user_id = ?", userId).
		First(&account).Error
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("credit account not found for user %d", userId)
		}
		return err
	}

	if account.Balance < amount {
		tx.Rollback()
		return fmt.Errorf("insufficient credit balance: have %d, need %d", account.Balance, amount)
	}

	err = tx.Model(&account).Updates(map[string]interface{}{
		"balance":     gorm.Expr("balance - ?", amount),
		"total_spent": gorm.Expr("total_spent + ?", amount),
		"updated_at":  common.GetTimestamp(),
	}).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// RecordCreditTransaction appends a ledger entry. Does NOT modify the balance —
// call IncreaseCreditBalance / DecreaseCreditBalance separately.
func RecordCreditTransaction(userId int, txType int, amount int64, relatedId int, description string) error {
	// Read current balance for snapshot
	balance, err := GetCreditBalance(userId)
	if err != nil {
		return err
	}

	record := &CreditTransaction{
		UserId:      userId,
		Type:        txType,
		Amount:      amount,
		Balance:     balance,
		RelatedId:   relatedId,
		Description: description,
		CreatedAt:   common.GetTimestamp(),
	}
	return DB.Create(record).Error
}

// GetCreditTransactions returns paginated credit transactions for a user.
func GetCreditTransactions(userId int, startIdx int, num int) ([]CreditTransaction, int64, error) {
	var transactions []CreditTransaction
	var total int64

	tx := DB.Model(&CreditTransaction{}).Where("user_id = ?", userId)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Order("id desc").Limit(num).Offset(startIdx).Find(&transactions).Error; err != nil {
		return nil, 0, err
	}
	return transactions, total, nil
}
