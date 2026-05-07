package mc

import (
	"encoding/json"
	"go-aliyunmc/utils"
	"log"
	"os"
	"strings"
)

type enOrZh struct {
	EnglishName string
	ChineseName string
}

func init() {
	testing := strings.HasSuffix(os.Args[0], ".test")
	if testing {
		if err := os.Chdir(utils.ProjectRoot()); err != nil {
			log.Fatal(err)
		}
	}

	Advancements = make(map[string]Advancement)

	var langEn map[string]string
	var langZh map[string]string

	langEnRaw, err := os.ReadFile("minecraft_en_us.json")

	if err != nil {
		log.Fatal(err)
	}

	langZhRaw, err := os.ReadFile("minecraft_zh_cn.json")

	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(langEnRaw, &langEn)

	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(langZhRaw, &langZh)

	if err != nil {
		log.Fatal(err)
	}

	buildAdvancements(langEn, langZh)
	buildBlocksAndItems(langEn, langZh)
	buildBiomes(langEn, langZh)
	buildEntities(langEn, langZh)
}
