package monitors

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/handlers/server"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/mcstatus-io/mcutil/v4/status"
)

func ServerStatusMonitor() {
	var activeInstance *store.Instance
	var ctx context.Context
	var cancel context.CancelFunc
	var err error

	logger := log.New(os.Stdout, "[ServerStatusMonitor] ", log.LstdFlags)
	logger.Println("Starting server status monitor")

	var initialLoadServerInfo = false

	for {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)

		activeInstance = store.GetActiveInstance()

		if activeInstance == nil {
			globals.IsServerRunning = false
			goto end
		}

		globals.ServerStatus, err = status.Modern(ctx, activeInstance.Ip, 25565)

		if err != nil {
			globals.IsServerRunning = false
			goto end
		}

		globals.IsServerRunning = true

		if !initialLoadServerInfo {
			initialLoadServerInfo = true
			_ = server.RefreshQueryFull(context.Background())
		}

	end:
		time.Sleep(5 * time.Second)
		cancel()
	}
}
