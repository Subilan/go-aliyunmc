package mc

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type enOrZh struct {
	EnglishName string `json:"englishName"`
	ChineseName string `json:"chineseName"`
}

var (
	dataOnce    sync.Once
	dataLoadErr error
)

// Advancements 返回成就翻译表；首次访问时自动加载语言数据。
func Advancements() map[string]Advancement {
	MustLoadData()
	return advancements
}

// Biomes 返回生物群系翻译表；首次访问时自动加载语言数据。
func Biomes() map[string]Biome {
	MustLoadData()
	return biomes
}

// BlocksAndItems 返回方块和物品翻译表；首次访问时自动加载语言数据。
func BlocksAndItems() map[string]BlockOrItem {
	MustLoadData()
	return blocksAndItems
}

// Entities 返回实体翻译表；首次访问时自动加载语言数据。
func Entities() map[string]Entity {
	MustLoadData()
	return entities
}

// Stats 返回统计信息翻译表；首次访问时自动加载语言数据。
func Stats() map[string]Stat {
	MustLoadData()
	return stats
}

// LoadData 加载 Minecraft 语言数据并构建翻译表。可以在测试和运行时显式调用。
func LoadData() error {
	dataOnce.Do(func() {
		dataLoadErr = loadData()
	})
	return dataLoadErr
}

// MustLoadData 加载 Minecraft 语言数据，失败时直接终止程序。
func MustLoadData() {
	if err := LoadData(); err != nil {
		log.Fatalf("加载 Minecraft 语言数据失败: %v", err)
	}
}

func loadData() error {
	advancements = make(map[string]Advancement)
	biomes = make(map[string]Biome)
	blocksAndItems = make(map[string]BlockOrItem)
	entities = make(map[string]Entity)
	stats = make(map[string]Stat)

	var langEn map[string]string
	var langZh map[string]string

	langEnRaw, err := os.ReadFile("minecraft_en_us.json")
	if err != nil {
		return err
	}

	langZhRaw, err := os.ReadFile("minecraft_zh_cn.json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(langEnRaw, &langEn)
	if err != nil {
		return err
	}

	err = json.Unmarshal(langZhRaw, &langZh)
	if err != nil {
		return err
	}

	buildAdvancements(langEn, langZh)
	buildBlocksAndItems(langEn, langZh)
	buildBiomes(langEn, langZh)
	buildEntities(langEn, langZh)
	buildStats(langEn, langZh)
	return nil
}
