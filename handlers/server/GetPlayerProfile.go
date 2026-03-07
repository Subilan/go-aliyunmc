package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/monitors"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type EssentialsData struct {
	Timestamps     map[string]int64 `yaml:"timestamps" json:"timestamps,omitempty"`
	LogoutLocation *LocationData    `yaml:"logoutlocation" json:"logoutLocation,omitempty"`
	LoginLocation  *LocationData    `yaml:"lastlocation" json:"loginLocation,omitempty"`
}

type LocationData struct {
	World     string  `yaml:"world" json:"world,omitempty"`
	WorldName string  `yaml:"world-name" json:"worldName,omitempty"`
	X         float64 `yaml:"x" json:"x,omitempty"`
	Y         float64 `yaml:"y" json:"y,omitempty"`
	Z         float64 `yaml:"z" json:"z,omitempty"`
	Yaw       float64 `yaml:"yaw" json:"yaw,omitempty"`
	Pitch     float64 `yaml:"pitch" json:"pitch,omitempty"`
}

type GetPlayerProfileResponse struct {
	UUID       string                      `json:"uuid"`
	GameName   string                      `json:"gameName"`
	Essentials *EssentialsData             `json:"essentials,omitempty"`
	Stats      map[string]map[string]int64 `json:"stats,omitempty"`
	PlayTime   *store.PlayTimeData         `json:"playTime,omitempty"`
}

func HandleGetPlayerProfile() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		gameName := c.Param("gameName")
		if gameName == "" {
			return nil, &helpers.HttpError{Code: 400, Details: "gameName is required"}
		}

		// 从内存中读取白名单查找 UUID
		whitelist := monitors.SnapshotWhitelist()

		var uuid string
		for _, item := range whitelist {
			if item.Name == gameName {
				uuid = item.Uuid
				break
			}
		}

		if uuid == "" {
			return nil, &helpers.HttpError{Code: 404, Details: "player not found in whitelist"}
		}

		response := GetPlayerProfileResponse{
			UUID:     uuid,
			GameName: gameName,
		}

		// 加载 Essentials 数据
		essentials, err := loadEssentials(uuid)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			// 文件不存在，继续
		} else {
			response.Essentials = essentials
		}

		// 加载 Stats 数据
		stats, err := loadStats(uuid)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			// 文件不存在，继续
		} else {
			response.Stats = stats
		}

		// 加载 PlayTime 数据
		playTime, err := store.GetPlayTimeByUUID(uuid)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
			// 记录不存在，继续
		} else {
			response.PlayTime = playTime
		}

		// 检查是否所有数据都缺失
		if response.Essentials == nil && response.Stats == nil && response.PlayTime == nil {
			return nil, &helpers.HttpError{Code: 404, Details: "no player data found"}
		}

		return helpers.Data(response), nil
	})
}

func loadEssentials(uuid string) (*EssentialsData, error) {
	filePath := filepath.Join(config.Cfg.Monitor.PlayerProfile.LocalEssentialsPath, uuid+".yml")

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var data EssentialsData
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func loadStats(uuid string) (map[string]map[string]int64, error) {
	filePath := filepath.Join(config.Cfg.Monitor.PlayerProfile.LocalStatsPath, uuid+".json")

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, err
	}

	statsRaw, ok := raw["stats"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid stats format")
	}

	result := make(map[string]map[string]int64)
	for key, value := range statsRaw {
		if innerMap, ok := value.(map[string]interface{}); ok {
			inner := make(map[string]int64)
			for k, v := range innerMap {
				if num, ok := v.(float64); ok {
					inner[k] = int64(num)
				}
			}
			result[key] = inner
		}
	}

	return result, nil
}
