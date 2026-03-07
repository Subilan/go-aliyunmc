package store

import (
	"database/sql"
	"errors"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers/db"
)

// UserPreference 表示用户的偏好设置
type UserPreference struct {
	Username string                   `json:"username"`
	Key      consts.UserPreferenceKey `json:"key"`
	Value    string                   `json:"value"`
}

// SetUserPreference 设置用户的偏好设置
// 如果记录已存在则更新，不存在则插入
func SetUserPreference(username string, key consts.UserPreferenceKey, value string) error {
	_, err := db.Pool.Exec(`
		INSERT INTO user_preferences (username, preference_key, preference_value)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE preference_value = ?
	`, username, key, value, value)

	return err
}

// GetUserPreference 获取用户的偏好设置
// 如果记录不存在，返回 sql.ErrNoRows 错误
func GetUserPreference(username string, key consts.UserPreferenceKey) (*UserPreference, error) {
	var result UserPreference

	err := db.Pool.QueryRow(`
		SELECT username, preference_key, preference_value
		FROM user_preferences
		WHERE username = ? AND preference_key = ?
	`, username, key).Scan(&result.Username, &result.Key, &result.Value)

	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetUserPreferenceBool 获取用户的布尔类型偏好设置
// 如果记录不存在，返回 sql.ErrNoRows 错误
// value 值应为 "true" 或 "false"
func GetUserPreferenceBool(username string, key consts.UserPreferenceKey) (bool, error) {
	pref, err := GetUserPreference(username, key)
	if err != nil {
		return false, err
	}

	return pref.Value == "true", nil
}

// DeleteUserPreference 删除用户的偏好设置
func DeleteUserPreference(username string, key consts.UserPreferenceKey) error {
	_, err := db.Pool.Exec(`
		DELETE FROM user_preferences
		WHERE username = ? AND preference_key = ?
	`, username, key)

	return err
}

// DeleteAllUserPreferences 删除用户的所有偏好设置
func DeleteAllUserPreferences(username string) error {
	_, err := db.Pool.Exec(`
		DELETE FROM user_preferences
		WHERE username = ?
	`, username)

	return err
}

// GetAllUserPreferences 获取用户的所有偏好设置
func GetAllUserPreferences(username string) ([]*UserPreference, error) {
	rows, err := db.Pool.Query(`
		SELECT username, preference_key, preference_value
		FROM user_preferences
		WHERE username = ?
		ORDER BY preference_key
	`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*UserPreference
	for rows.Next() {
		var pref UserPreference
		if err := rows.Scan(&pref.Username, &pref.Key, &pref.Value); err != nil {
			return nil, err
		}
		result = append(result, &pref)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// HasUserPreference 检查用户是否有某个偏好设置记录
func HasUserPreference(username string, key consts.UserPreferenceKey) bool {
	var count int
	err := db.Pool.QueryRow(`
		SELECT COUNT(*)
		FROM user_preferences
		WHERE username = ? AND preference_key = ?
	`, username, key).Scan(&count)

	if err != nil {
		return false
	}

	return count > 0
}

// SetUserPreferenceBool 设置用户的布尔类型偏好设置
func SetUserPreferenceBool(username string, key consts.UserPreferenceKey, value bool) error {
	strValue := "false"
	if value {
		strValue = "true"
	}
	return SetUserPreference(username, key, strValue)
}

// GetUserPreferenceOrDefault 获取用户的偏好设置，如果不存在则返回默认值
func GetUserPreferenceOrDefault(username string, key consts.UserPreferenceKey, defaultValue string) string {
	pref, err := GetUserPreference(username, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultValue
		}
		return defaultValue
	}

	return pref.Value
}

// GetUserPreferenceBoolOrDefault 获取用户的布尔类型偏好设置，如果不存在则返回默认值
func GetUserPreferenceBoolOrDefault(username string, key consts.UserPreferenceKey, defaultValue bool) bool {
	pref, err := GetUserPreference(username, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultValue
		}
		return defaultValue
	}

	return pref.Value == "true"
}
