package mc

import (
	"regexp"
	"strings"
)

type Stat struct {
	enOrZh
}

// stats 保存统计信息翻译表，由 LoadData 填充。
var stats map[string]Stat

var statRe = regexp.MustCompile(`stat\.minecraft\.([A-Za-z_0-9]+).*`)

func buildStats(langEn, langZh map[string]string) {
	for key, value := range langEn {
		if !strings.HasPrefix(key, "stat.") {
			continue
		}

		matches := statRe.FindStringSubmatch(key)

		if len(matches) < 2 {
			continue
		}

		statIdentifier := "minecraft:" + matches[1]

		zhName, ok := langZh[key]

		if !ok {
			zhName = value
		}

		stats[statIdentifier] = Stat{
			enOrZh: enOrZh{
				EnglishName: value,
				ChineseName: zhName,
			},
		}
	}
}
