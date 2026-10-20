package mc

import (
	"regexp"
	"strings"
)

type BlockOrItem struct {
	enOrZh
}

// blocksAndItems 保存方块和物品翻译表，由 LoadData 填充。
var blocksAndItems map[string]BlockOrItem

var blockAndItemsRe = regexp.MustCompile(`(block|item)\.minecraft\.([A-Za-z_0-9]+).*`)

func buildBlocksAndItems(langEn, langZh map[string]string) {
	for key, value := range langEn {
		if !strings.HasPrefix(key, "block.") && !strings.HasPrefix(key, "item.") {
			continue
		}

		matches := blockAndItemsRe.FindStringSubmatch(key)

		if len(matches) < 3 {
			continue
		}

		blockOrItemIdentifier := "minecraft:" + matches[2]

		zhName, ok := langZh[key]
		if !ok {
			zhName = value
		}

		blocksAndItems[blockOrItemIdentifier] = BlockOrItem{
			enOrZh: enOrZh{
				EnglishName: value,
				ChineseName: zhName,
			},
		}
	}
}
