package store

import (
	"errors"
	"go-aliyunmc/store/models"

	"gorm.io/gorm"
)

// GetActiveInstance 获取当前活跃的实例，即目前存在的那一个实例，如果没有则返回 nil。
// 注意，这个函数假设系统中最多只能有一个活跃实例。如果数据库中存在多条实例记录，这个函数将返回第一条记录。
// 如果没有活跃的实例，该函数不会返回错误。
func GetActiveInstance() (*models.Instance, error) {
	var instance models.Instance
	err := DB.First(&instance).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &instance, nil
}

func DeleteActiveInstance() error {
	var instance models.Instance
	err := DB.First(&instance).Error

	if err != nil {
		return err
	}

	return DB.Delete(&instance).Error
}
