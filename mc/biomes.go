package mc

import "regexp"

type Biome struct {
	enOrZh
}

// biomes 保存生物群系翻译表，由 LoadData 填充。
var biomes map[string]Biome

var biomesRe = regexp.MustCompile(`biome\.minecraft\.([A-Za-z_0-9]+).*`)

func buildBiomes(langEn, langZh map[string]string) {
	for key, value := range langEn {
		if !biomesRe.MatchString(key) {
			continue
		}

		matches := biomesRe.FindStringSubmatch(key)

		if len(matches) < 2 {
			continue
		}

		biomeIdentifier := "minecraft:" + matches[1]

		zhName, ok := langZh[key]

		if !ok {
			zhName = value
		}

		biomes[biomeIdentifier] = Biome{
			enOrZh: enOrZh{
				EnglishName: value,
				ChineseName: zhName,
			},
		}
	}
}
