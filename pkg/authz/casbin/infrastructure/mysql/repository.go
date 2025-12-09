package mysql

import (
	"context"

	"gorm.io/gorm"

	"github.com/kart-io/sentinel-x/pkg/authz/casbin"
)

// CasbinRule represents the database model for Casbin policies
type CasbinRule struct {
	ID    uint   `gorm:"primaryKey;autoIncrement"`
	PType string `gorm:"size:100;index"`
	V0    string `gorm:"size:100;index"`
	V1    string `gorm:"size:100;index"`
	V2    string `gorm:"size:100;index"`
	V3    string `gorm:"size:100;index"`
	V4    string `gorm:"size:100;index"`
	V5    string `gorm:"size:100;index"`
}

// TableName sets the table name
func (CasbinRule) TableName() string {
	return "casbin_rule"
}

type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new MySQL repository
func NewRepository(db *gorm.DB) (*Repository, error) {
	// Auto migrate the table
	if err := db.AutoMigrate(&CasbinRule{}); err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
}

func (r *Repository) LoadPolicies(ctx context.Context) ([]*casbin.Policy, error) {
	var rules []CasbinRule
	if err := r.db.WithContext(ctx).Find(&rules).Error; err != nil {
		return nil, err
	}

	policies := make([]*casbin.Policy, len(rules))
	for i, rule := range rules {
		policies[i] = &casbin.Policy{
			PType: rule.PType,
			V0:    rule.V0,
			V1:    rule.V1,
			V2:    rule.V2,
			V3:    rule.V3,
			V4:    rule.V4,
			V5:    rule.V5,
		}
	}
	return policies, nil
}

func (r *Repository) SavePolicies(ctx context.Context, policies []*casbin.Policy) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Truncate table (or delete all)
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&CasbinRule{}).Error; err != nil {
			return err
		}

		if len(policies) == 0 {
			return nil
		}

		rules := make([]CasbinRule, len(policies))
		for i, p := range policies {
			rules[i] = CasbinRule{
				PType: p.PType,
				V0:    p.V0,
				V1:    p.V1,
				V2:    p.V2,
				V3:    p.V3,
				V4:    p.V4,
				V5:    p.V5,
			}
		}
		return tx.Create(&rules).Error
	})
}

func (r *Repository) AddPolicy(ctx context.Context, p *casbin.Policy) error {
	rule := CasbinRule{
		PType: p.PType,
		V0:    p.V0,
		V1:    p.V1,
		V2:    p.V2,
		V3:    p.V3,
		V4:    p.V4,
		V5:    p.V5,
	}
	return r.db.WithContext(ctx).Create(&rule).Error
}

func (r *Repository) RemovePolicy(ctx context.Context, p *casbin.Policy) error {
	rule := CasbinRule{
		PType: p.PType,
		V0:    p.V0,
		V1:    p.V1,
		V2:    p.V2,
		V3:    p.V3,
		V4:    p.V4,
		V5:    p.V5,
	}
	return r.db.WithContext(ctx).Where(&rule).Delete(&CasbinRule{}).Error
}

func (r *Repository) RemoveFilteredPolicy(ctx context.Context, ptype string, fieldIndex int, fieldValues ...string) error {
	query := r.db.WithContext(ctx).Model(&CasbinRule{}).Where("p_type = ?", ptype)

	if fieldIndex <= 0 && 0 < fieldIndex+len(fieldValues) {
		query = query.Where("v0 = ?", fieldValues[0-fieldIndex])
	}
	if fieldIndex <= 1 && 1 < fieldIndex+len(fieldValues) {
		query = query.Where("v1 = ?", fieldValues[1-fieldIndex])
	}
	if fieldIndex <= 2 && 2 < fieldIndex+len(fieldValues) {
		query = query.Where("v2 = ?", fieldValues[2-fieldIndex])
	}
	if fieldIndex <= 3 && 3 < fieldIndex+len(fieldValues) {
		query = query.Where("v3 = ?", fieldValues[3-fieldIndex])
	}
	if fieldIndex <= 4 && 4 < fieldIndex+len(fieldValues) {
		query = query.Where("v4 = ?", fieldValues[4-fieldIndex])
	}
	if fieldIndex <= 5 && 5 < fieldIndex+len(fieldValues) {
		query = query.Where("v5 = ?", fieldValues[5-fieldIndex])
	}

	return query.Delete(&CasbinRule{}).Error
}
