package monitors

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/filelog"
	"github.com/Subilan/go-aliyunmc/helpers/commands"
	"github.com/Subilan/go-aliyunmc/helpers/store"
)

var whitelist []store.WhitelistItem

func Whitelist(quit chan bool) {
	logger := filelog.NewLogger("whitelist", "Whitelist")
	logger.Println("starting...")
	ticker := time.NewTicker(config.Cfg.Monitor.Whitelist.IntervalDuration())

	cmd := commands.MustGetCommand(consts.CmdTypeGetWhitelist)

	cacheFileContent, err := os.ReadFile(config.Cfg.Monitor.Whitelist.CacheFile)

	if err != nil {
		logger.Println("cannot read whitelist cache file")
	} else {
		err = json.Unmarshal(cacheFileContent, &whitelist)

		if err != nil {
			logger.Println("cannot parse whitelist cache file")
		} else {
			logger.Printf("loaded whitelist cache file with %d records", len(whitelist))
		}
	}

	for {
		func() {
			activeInstance, err := store.GetDeployedActiveInstance()

			if err != nil {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), config.Cfg.Monitor.Whitelist.TimeoutDuration())
			defer cancel()
			logger.Println("refreshing...")

			output, err := cmd.RunWithoutCooldown(ctx, *activeInstance.Ip, nil, nil)

			if err != nil {
				logger.Println("cannot get whitelist: " + err.Error())
				return
			}

			var result = make([]store.WhitelistItem, 0, 10)

			if err := json.Unmarshal([]byte(output), &result); err != nil {
				logger.Println("cannot unmarshal whitelist: " + err.Error())
				return
			}

			isSame := false

			if len(result) == len(whitelist) {
				isSame = true

				sort.SliceStable(result, func(i, j int) bool {
					return result[i].Name < result[j].Name
				})

				sort.SliceStable(whitelist, func(i, j int) bool {
					return whitelist[i].Uuid < whitelist[j].Uuid
				})

				resultPtr := 0
				whitelistPtr := 0
				length := len(result)

				for resultPtr < length && whitelistPtr < length {
					if result[whitelistPtr].Name != whitelist[whitelistPtr].Name {
						isSame = false
						break
					}
					resultPtr++
					whitelistPtr++
				}
			}

			if !isSame {
				logger.Println("whitelist has changed")

				whitelist = result
				logger.Println("updating cache file")
				err = os.WriteFile(config.Cfg.Monitor.Whitelist.CacheFile, []byte(output), 0644)

				if err != nil {
					logger.Println("cannot write whitelist: " + err.Error())
				} else {
					logger.Println("ok")
				}
			} else {
				logger.Println("whitelist remains unchanged")
			}
		}()

		select {
		case <-ticker.C:
			continue
		case <-quit:
			return
		}
	}
}
