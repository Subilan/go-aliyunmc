package mc

import (
	"regexp"
	"strings"
)

type Stat struct {
	enOrZh
}

var Stats = make(map[string]Stat)

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

		Stats[statIdentifier] = Stat{
			enOrZh: enOrZh{
				EnglishName: value,
				ChineseName: zhName,
			},
		}
	}
}
