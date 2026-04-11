package store

import (
	"errors"
	"go-aliyunmc/store/models"

	"gorm.io/gorm"
)

// GetActiveInstanceDefaultNil 获取当前活跃的实例，如果没有则返回 nil, nil
func GetActiveInstanceDefaultNil() (*models.Instance, error) {
	instance, err := GetActiveInstance()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return instance, err
}

// HasActiveInstance 检查是否存在活跃实例，存在返回 true，否则返回 false。
// 注意如果数据库查询出现错误，此函数也会返回 false。
func HasActiveInstance() bool {
	instance, err := GetActiveInstanceDefaultNil()

	if err != nil {
		return false
	}

	return instance != nil
}

// GetActiveInstance 获取当前活跃的实例，如果没有则返回错误。
// 注意，这个函数假设系统中最多只能有一个活跃实例。如果数据库中存在多条实例记录，这个函数将返回第一条记录。
func GetActiveInstance() (*models.Instance, error) {
	var instance models.Instance
	err := Silent().First(&instance).Error
	if err != nil {
		return nil, err
	}

	return &instance, nil
}

// DeleteActiveInstance 删除当前活跃的实例，如果没有则返回错误。
func DeleteActiveInstance() error {
	instance, err := GetActiveInstance()
	if err != nil {
		return err
	}

	return DB.Delete(instance).Error
}

// GetActiveInstanceIpNonEmpty 获取当前活跃实例的 IP 地址，如果没有活跃实例或 IP 地址为空，则返回错误。
func GetActiveInstanceIpNonEmpty() (string, error) {
	instance, err := GetActiveInstance()

	if err != nil {
		return "", err
	}

	if instance.Ip == "" {
		return "", errors.New("IP地址未分配")
	}

	return instance.Ip, nil
}

// SetActiveDeployed 将当前活跃实例的 IsDeployed 字段设置为 true，表示该实例已部署完成。如果没有活跃实例，则返回错误。
func SetActiveDeployed() error {
	instance, err := GetActiveInstance()

	if err != nil {
		return err
	}

	instance.IsDeployed = true
	return DB.Save(instance).Error
}
