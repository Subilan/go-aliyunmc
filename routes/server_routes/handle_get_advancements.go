package server_routes

import (
	"encoding/json"
	"fmt"
	"github.com/Subilan/go-aliyunmc/context_util"
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/mc"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
)

const advancementsDir = "remote_data_cache/player_advancements"

type rawAchievement struct {
	Criteria map[string]string `json:"criteria"`
	Done     bool              `json:"done"`
}

type AdvancementEntry struct {
	mc.Advancement
	Done     bool                 `json:"done"`
	Criteria map[string]time.Time `json:"criteria"`
}

var advancementKeyRe = regexp.MustCompile(`minecraft:(adventure|story|end|nether|husbandry).*`)

func HandleGetAdvancements(c *gin.Context) (any, error) {
	uuid := c.Param("uuid")
	if uuid == "" {
		return nil, h.HttpError(http.StatusBadRequest, "缺少玩家UUID参数")
	}

	// Check privacy
	var owner models.User
	if err := store.DB.Where("whitelist_uuid = ?", uuid).First(&owner).Error; err == nil {
		currentUser, _ := context_util.GetUser(c)
		if currentUser != nil && currentUser.ID != owner.ID {
			prefs, err := store.GetUserPreferences(owner.ID)
			if err == nil && prefs.DisallowPublicGameStats {
				return nil, h.HttpError(http.StatusForbidden, "该玩家不允许其他人查看游戏统计")
			}
		}
	}

	data, err := os.ReadFile(fmt.Sprintf("%s/%s.json", advancementsDir, uuid))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, h.HttpError(http.StatusNotFound, "未找到该玩家的成就数据")
		}
		return nil, fmt.Errorf("读取成就数据失败: %w", err)
	}

	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("解析成就数据失败: %w", err)
	}

	const timeLayout = "2006-01-02 15:04:05 -0700"

	playerAdvs := make(map[string]*AdvancementEntry)

	for key, rawValue := range rawData {
		if !advancementKeyRe.MatchString(key) {
			continue
		}

		var raw rawAchievement
		if err := json.Unmarshal(rawValue, &raw); err != nil {
			continue
		}

		adv, known := mc.Advancements[key]
		if !known {
			continue
		}

		criteria := make(map[string]time.Time, len(raw.Criteria))
		for k, v := range raw.Criteria {
			t, err := time.Parse(timeLayout, v)
			if err != nil {
				continue
			}
			criteria[k] = t
		}

		playerAdvs[key] = &AdvancementEntry{
			Advancement: adv,
			Done:        raw.Done,
			Criteria:    criteria,
		}
	}

	result := make([]AdvancementEntry, 0, len(mc.Advancements))
	for key, adv := range mc.Advancements {
		if existing, ok := playerAdvs[key]; ok {
			result = append(result, *existing)
		} else {
			result = append(result, AdvancementEntry{
				Advancement: adv,
				Done:        false,
				Criteria:    make(map[string]time.Time),
			})
		}
	}

	return result, nil
}
