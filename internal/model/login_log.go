package model

import "gorm.io/gorm"

// LoginLog 登录日志
type LoginLog struct {
	ID        uint64         `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	UserID    string         `json:"user_id" gorm:"size:32;index;comment:用户ID"`
	Username  string         `json:"username" gorm:"size:64;index;comment:用户名"`
	IP        string         `json:"ip" gorm:"size:64;comment:登录IP"`
	UserAgent string         `json:"user_agent" gorm:"size:255;comment:浏览器标识"`
	Status    int            `json:"status" gorm:"default:0;comment:登录状态 1成功 0失败"`
	Message   string         `json:"message" gorm:"size:255;comment:失败原因"`
	CreatedAt int64          `json:"created_at" gorm:"autoCreateTime:milli;comment:创建时间(时间戳)"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:软删除时间"`
}

// TableName returns the table name for GORM.
func (l *LoginLog) TableName() string {
	return "login_logs"
}
