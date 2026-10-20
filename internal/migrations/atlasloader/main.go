// Command atlasloader prints the MySQL schema derived from the GORM models.
// It is executed by Atlas (see atlas.hcl) to plan versioned migrations.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Subilan/go-aliyunmc/internal/store/models"

	"ariga.io/atlas-provider-gorm/gormschema"
)

func main() {
	legacy := flag.Bool("legacy", false, "exclude task_output_lines (only for generating the pre-versioning baseline)")
	flag.Parse()

	entities := models.Entities()
	if *legacy {
		// 基线迁移生成于 task_output_lines 引入之前，需要排除该模型。
		filtered := entities[:0]
		for _, entity := range entities {
			if _, ok := entity.(*models.TaskOutputLine); ok {
				continue
			}
			filtered = append(filtered, entity)
		}
		entities = filtered
	}

	schema, err := gormschema.New("mysql").Load(entities...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Print(schema)
}
