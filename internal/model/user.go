package model

import (
	"time"

	"gorm.io/gorm"
)

// User represents the user model in the database.
type User struct {
	ID        uint64         `json:"id" gorm:"primaryKey;autoIncrement;comment:用户ID"`
	Username  string         `json:"username" gorm:"size:64;not null;uniqueIndex:uk_username;comment:用户名"`
	Email     *string        `json:"email" gorm:"size:128;uniqueIndex:uk_email;comment:邮箱"`
	Password  string         `json:"-" gorm:"size:255;not null;comment:密码Hash"`
	Avatar    string         `json:"avatar" gorm:"size:255;comment:头像URL"`
	Mobile    string         `json:"mobile" gorm:"size:20;index:idx_mobile;comment:手机号"`
	Status    int            `json:"status" gorm:"default:1;index:idx_status;comment:状态 1启用 0禁用"`
	CreatedAt int64          `json:"created_at" gorm:"autoCreateTime:milli;comment:创建时间(时间戳)"`
	UpdatedAt int64          `json:"updated_at" gorm:"autoUpdateTime:milli;comment:更新时间(时间戳)"`
	CreatedBy uint64         `json:"created_by" gorm:"default:0;comment:创建人"`
	UpdatedBy uint64         `json:"updated_by" gorm:"default:0;comment:更新人"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:软删除时间"`
}

// UserList contains a list of users and pagination info.
type UserList struct {
	TotalCount int64   `json:"totalCount"`
	Items      []*User `json:"items"`
}

// TableName returns the table name for GORM.
func (u *User) TableName() string {
	return "users"
}

// BeforeCreate sets the CreatedAt and UpdatedAt fields.
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	now := time.Now().UnixMilli()
	u.CreatedAt = now
	u.UpdatedAt = now
	return
}

// BeforeUpdate sets the UpdatedAt field.
func (u *User) BeforeUpdate(tx *gorm.DB) (err error) {
	u.UpdatedAt = time.Now().UnixMilli()
	return
}
