package mc

import (
	"regexp"
	"strings"
)

// Advancement 表示 Minecraft Java Edition 中的一个成就，包括资源位置、英文名称、中文名称、英文描述和中文描述。
type Advancement struct {
	enOrZh
	// ResourceLocation 是该成就的资源位置，其作用类似于标识符，例如 "minecraft:story/mine_stone"。
	ResourceLocation string
	// EnglishDescription 是该成就的英文描述，例如 "Mine Stone with your new Pickaxe"。
	EnglishDescription string
	// ChineseDescription 是该成就的中文描述，例如 "用你的新镐挖掘石头"。
	ChineseDescription string
}

// Advancements 是一个映射，键是成就的资源位置，值是对应的 Advancement 结构体实例，包含了 Minecraft Java Edition 26.1 中所有成就的信息。
var Advancements = make(map[string]Advancement)

var advancementRe = regexp.MustCompile(`advancements\.(story|nether|end|husbandry|adventure)\.([A-Za-z_0-9]+)\.(title|description)`)

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

		Advancements[resourceLocation] = Advancement{
			ResourceLocation: resourceLocation,
			enOrZh: enOrZh{
				EnglishName: enTitle,
				ChineseName: zhTitle,
			},
			EnglishDescription: enDesc,
			ChineseDescription: zhDesc,
		}
	}
}
