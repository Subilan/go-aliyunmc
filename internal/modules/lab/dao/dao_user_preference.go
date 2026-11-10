package dao

import (
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/models"
)

// GetUserPreferences 获取用户的偏好设置，无记录时返回默认值
func GetUserPreferences(userID uint) (labmodels.Preferences, error) {
	var pref labmodels.UserPreference
	err := store.DB.Where("user_id = ?", userID).First(&pref).Error
	if err != nil {
		return labmodels.DefaultPreferences, nil
	}
	return pref.ParsePreferences()
}

// SetUserPreferences 保存用户的偏好设置（存在则更新，不存在则创建）
func SetUserPreferences(userID uint, prefs labmodels.Preferences) error {
	var pref labmodels.UserPreference
	err := store.DB.Where("user_id = ?", userID).First(&pref).Error
	if err != nil {
		pref = labmodels.UserPreference{UserID: userID}
	}
	if err := pref.SetPreferences(prefs); err != nil {
		return err
	}
	return store.DB.Save(&pref).Error
}
