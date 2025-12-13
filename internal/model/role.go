package model

import (
	"time"

	"gorm.io/gorm"
)

// Role represents a user role in the system.
type Role struct {
	ID          uint64         `json:"id" gorm:"primaryKey;autoIncrement;comment:角色ID"`
	Code        string         `json:"code" gorm:"size:32;not null;uniqueIndex:uk_code;comment:角色编码"`
	Name        string         `json:"name" gorm:"size:64;not null;comment:角色名称"`
	Description string         `json:"description" gorm:"size:255;comment:描述"`
	Status      int            `json:"status" gorm:"default:1;index:idx_status;comment:状态 1启用 0禁用"`
	CreatedAt   int64          `json:"created_at" gorm:"autoCreateTime:milli;comment:创建时间"`
	UpdatedAt   int64          `json:"updated_at" gorm:"autoUpdateTime:milli;comment:更新时间"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index;comment:软删除时间"`
}

// TableName returns the table name for GORM.
func (r *Role) TableName() string {
	return "roles"
}

// BeforeCreate sets the CreatedAt and UpdatedAt fields.
func (r *Role) BeforeCreate(tx *gorm.DB) (err error) {
	now := time.Now().UnixMilli()
	r.CreatedAt = now
	r.UpdatedAt = now
	return
}

// BeforeUpdate sets the UpdatedAt field.
func (r *Role) BeforeUpdate(tx *gorm.DB) (err error) {
	r.UpdatedAt = time.Now().UnixMilli()
	return
}

// UserRole represents the many-to-many relationship between users and roles.
type UserRole struct {
	ID        uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint64 `json:"user_id" gorm:"uniqueIndex:uk_user_role;index:idx_user_id;not null;comment:用户ID"`
	RoleID    uint64 `json:"role_id" gorm:"uniqueIndex:uk_user_role;index:idx_role_id;not null;comment:角色ID"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime:milli;comment:创建时间"`
}

// TableName returns the table name for GORM.
func (ur *UserRole) TableName() string {
	return "user_roles"
}

// BeforeCreate sets the CreatedAt field.
func (ur *UserRole) BeforeCreate(_ *gorm.DB) (err error) {
	ur.CreatedAt = time.Now().UnixMilli()
	return
}
