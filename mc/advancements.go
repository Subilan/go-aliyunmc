package mc

import (
	"regexp"
	"strings"
)

// Advancement 表示 Minecraft Java Edition 中的一个成就，包括资源位置、英文名称、中文名称、英文描述和中文描述。
type Advancement struct {
	enOrZh
	// ResourceLocation 是该成就的资源位置，其作用类似于标识符，例如 "minecraft:story/mine_stone"。
	ResourceLocation string `json:"resourceLocation"`
	// EnglishDescription 是该成就的英文描述，例如 "Mine Stone with your new Pickaxe"。
	EnglishDescription string `json:"englishDescription"`
	// ChineseDescription 是该成就的中文描述，例如 "用你的新镐挖掘石头"。
	ChineseDescription string `json:"chineseDescription"`
	// IsGoal 表示该成就是否是一个目标成就。
	IsGoal bool `json:"isGoal"`
	// IsChallenge 表示该成就是否是一个挑战成就。
	IsChallenge bool `json:"isChallenge"`
}

// advancements 保存成就翻译表，由 LoadData 填充。
var advancements map[string]Advancement

var advancementRe = regexp.MustCompile(`advancements\.(story|nether|end|husbandry|adventure)\.([A-Za-z_0-9]+)\.(title|description)`)

var goalAdvancementNames = map[string]struct{}{
	"Zombie Doctor":       {},
	"Beaconator":          {},
	"The Next Generation": {},
	"The End... Again...": {},
	"You Need a Mint":     {},
	"Sky's the Limit":     {},
	"Postmortal":          {},
	"Mob Kabob":           {},
	"Hired Help":          {},
	"Revaulting":          {},
}

var challengeAdvancementNames = map[string]struct{}{
	"Return to Sender":          {},
	"Subspace Bubble":           {},
	"Uneasy Alliance":           {},
	"Cover Me in Debris":        {},
	"Hot Tourist Destinations":  {},
	"A Furious Cocktail":        {},
	"How Did We Get Here?":      {},
	"Great View From Up Here":   {},
	"Hero of the Village":       {},
	"It Spreads":                {},
	"Monsters Hunted":           {},
	"Smithing with Style":       {},
	"Two Birds, One Arrow":      {},
	"Arbalistic":                {},
	"Adventuring Time":          {},
	"Sniper Duel":               {},
	"Bullseye":                  {},
	"Blowback":                  {},
	"Over-Overkill":             {},
	"Two By Two":                {},
	"A Complete Catalogue":      {},
	"A Balanced Diet":           {},
	"Serious Dedication":        {},
	"With Our Powers Combined!": {},
	"The Whole Pack":            {},
}

func IsGoalAdvancement(name string) bool {
	_, ok := goalAdvancementNames[name]
	return ok
}

func IsChallengeAdvancement(name string) bool {
	_, ok := challengeAdvancementNames[name]
	return ok
}

func buildAdvancements(langEn, langZh map[string]string) {
	var keyResourceLocationMap = make(map[string]string)

	for key := range langEn {
		if !strings.HasPrefix(key, "advancements.") || !strings.HasSuffix(key, ".title") && !strings.HasSuffix(key, ".description") {
			continue
		}

		sansLast := key[:strings.LastIndex(key, ".")]
		matches := advancementRe.FindStringSubmatch(key)

		if len(matches) < 4 {
			continue
		}

		identifier := "minecraft:" + matches[1] + "/" + matches[2]

		keyResourceLocationMap[sansLast] = identifier
	}

	for key, resourceLocation := range keyResourceLocationMap {
		enTitle := langEn[key+".title"]
		enDesc := langEn[key+".description"]
		zhTitle, ok := langZh[key+".title"]
		if !ok {
			zhTitle = enTitle
		}
		zhDesc, ok := langZh[key+".description"]
		if !ok {
			zhDesc = enDesc
		}

		advancements[resourceLocation] = Advancement{
			ResourceLocation: resourceLocation,
			enOrZh: enOrZh{
				EnglishName: enTitle,
				ChineseName: zhTitle,
			},
			EnglishDescription: enDesc,
			ChineseDescription: zhDesc,
			IsGoal:             IsGoalAdvancement(enTitle),
			IsChallenge:        IsChallengeAdvancement(enTitle),
		}
	}
}
