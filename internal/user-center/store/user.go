package store

import (
	"context"
	stderrors "errors"

	"gorm.io/gorm"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/pkg/store"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

type users struct {
	db *gorm.DB
}

func newUsers(db *gorm.DB) *users {
	return &users{db}
}

// Create 创建新用户
func (u *users) Create(ctx context.Context, user *model.User) error {
	if err := u.db.WithContext(ctx).Create(user).Error; err != nil {
		// 检查唯一键冲突
		if stderrors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.ErrAlreadyExists.WithMessage("用户名或邮箱已存在")
		}
		return errors.ErrDatabase.WithCause(err)
	}
	return nil
}

// Update 更新现有用户
func (u *users) Update(ctx context.Context, user *model.User) error {
	result := u.db.WithContext(ctx).Save(user)
	if result.Error != nil {
		return errors.ErrDatabase.WithCause(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.ErrUserNotFound
	}
	return nil
}

// Delete 根据用户名删除用户
func (u *users) Delete(ctx context.Context, username string) error {
	result := u.db.WithContext(ctx).Where("username = ?", username).Delete(&model.User{})
	if result.Error != nil {
		return errors.ErrDatabase.WithCause(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.ErrUserNotFound
	}
	return nil
}

// Get 根据用户名检索用户
func (u *users) Get(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := u.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.ErrDatabase.WithCause(err)
	}
	return &user, nil
}

// GetByUserId 根据用户 ID 检索用户
func (u *users) GetByUserID(ctx context.Context, userID uint64) (*model.User, error) {
	var user model.User
	if err := u.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.ErrDatabase.WithCause(err)
	}
	return &user, nil
}

// List 使用窗口函数查询用户列表并返回总数，一次查询完成，避免 N+1 问题。
// 使用 COUNT(*) OVER() 窗口函数在单次查询中同时获取总记录数和分页数据。
// 明确指定查询字段，排除敏感的 password 字段。
//
// 参数:
//   - ctx: 上下文，用于超时控制和链路追踪
//   - offset: 分页偏移量
//   - limit: 每页数量
//
// 返回:
//   - int64: 符合条件的总记录数
//   - []*model.User: 当前页的用户列表（不包含 password 字段）
//   - error: 查询失败时返回的错误
func (u *users) List(ctx context.Context, opts ...store.Option) (int64, []*model.User, error) {
	var results []struct {
		model.User
		TotalCount int64 `gorm:"column:total_count"`
	}

	db := store.NewWhere(opts...).Where(u.db.WithContext(ctx))

	// 使用窗口函数 COUNT(*) OVER() 在单次查询中获取总数和分页数据
	// 明确指定字段列表，排除 password 和 deleted_at 等敏感字段
	err := db.
		Select(`
			id,
			username,
			email,
			avatar,
			mobile,
			status,
			created_at,
			updated_at,
			created_by,
			updated_by,
			COUNT(*) OVER() as total_count
		`).
		Model(&model.User{}).
		Find(&results).Error
	if err != nil {
		return 0, nil, errors.ErrDatabase.WithCause(err)
	}

	// 空结果直接返回
	if len(results) == 0 {
		return 0, []*model.User{}, nil
	}

	// 提取用户列表（不包含 TotalCount 字段）
	users := make([]*model.User, len(results))
	for i := range results {
		users[i] = &results[i].User
	}

	// 所有行的 total_count 相同，取第一行的值即可
	return results[0].TotalCount, users, nil
}
