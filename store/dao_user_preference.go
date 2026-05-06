package store

import "go-aliyunmc/store/models"

// GetUserPreferences 获取用户的偏好设置，无记录时返回默认值
func GetUserPreferences(userID uint) (models.Preferences, error) {
	var pref models.UserPreference
	err := DB.Where("user_id = ?", userID).First(&pref).Error
	if err != nil {
		return models.DefaultPreferences, nil
	}
	return pref.ParsePreferences()
}

// SetUserPreferences 保存用户的偏好设置（存在则更新，不存在则创建）
func SetUserPreferences(userID uint, prefs models.Preferences) error {
	var pref models.UserPreference
	err := DB.Where("user_id = ?", userID).First(&pref).Error
	if err != nil {
		pref = models.UserPreference{UserID: userID}
	}
	if err := pref.SetPreferences(prefs); err != nil {
		return err
	}
	return DB.Save(&pref).Error
}
