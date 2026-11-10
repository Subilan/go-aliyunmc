package mc

import "regexp"

type Entity struct {
	enOrZh
}

// entities 保存实体翻译表，由 LoadData 填充。
var entities map[string]Entity

var entityRe = regexp.MustCompile(`entity\.minecraft\.([A-Za-z_0-9]+).*`)

func buildEntities(langEn, langZh map[string]string) {
	for key, value := range langEn {
		if !entityRe.MatchString(key) {
			continue
		}

		matches := entityRe.FindStringSubmatch(key)

		if len(matches) < 2 {
			continue
		}

		entityIdentifier := "minecraft:" + matches[1]

		zhName, ok := langZh[key]

		if !ok {
			zhName = value
		}

		entities[entityIdentifier] = Entity{
			enOrZh: enOrZh{
				EnglishName: value,
				ChineseName: zhName,
			},
		}
	}
}
