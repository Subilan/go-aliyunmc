package instances

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/commands"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

const deleteInstanceTimeout = 15 * time.Second
const safeDeleteInstanceTimeout = 2 * time.Minute

type DeleteInstanceQuery struct {
	// Force 表示是否要强制删除实例。此选项只应该在实例上的重要资料已经备份的情况下指定。
	// 在底层，此选项表示阿里云SDK不检查实例的状态，并且强制对正在运行的实例执行断电操作。
	Force bool `form:"force"`

	// ArchiveAndForce 表示是否先归档再强制删除实例。
	// 此选项指定后，在删除实例之前会先尝试停止服务器并对其归档，如果未发生错误，就强制删除实例。
	// 其中，强制删除实例的环节相当于指定了 Force 为true，且只会在归档成功后执行。
	// 此选项只能在已经部署的实例上正常工作。
	ArchiveAndForce bool `form:"archiveAndForce"`
}

var deleteInstanceMutex sync.Mutex

func HandleDeleteInstance() gin.HandlerFunc {
	return helpers.QueryHandler[DeleteInstanceQuery](func(query DeleteInstanceQuery, c *gin.Context) (any, error) {
		ok := deleteInstanceMutex.TryLock()

		if !ok {
			return nil, &helpers.HttpError{Code: http.StatusServiceUnavailable, Details: "instance is being deleted"}
		}

		defer deleteInstanceMutex.Unlock()

		userId, err := gctx.ShouldGetUserId(c)

		if err != nil {
			return nil, err
		}

		var timeout time.Duration

		if query.ArchiveAndForce {
			timeout = safeDeleteInstanceTimeout
		} else {
			timeout = deleteInstanceTimeout
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		inst, err := store.GetIpAllocatedActiveInstance()

		if err != nil {
			return nil, err
		}

		if query.ArchiveAndForce {
			if !inst.Deployed {
				return nil, &helpers.HttpError{Code: http.StatusForbidden, Details: "instance is not deployed and cannot be archived"}
			}

			err = commands.StopAndArchiveServer(ctx, *inst.Ip, &userId, "delete instance")

			if err != nil {
				return nil, err
			}
		}

		err = helpers.DeleteInstance(ctx, inst.InstanceId, query.ArchiveAndForce || query.Force)

		if err != nil {
			return nil, err
		}

		return gin.H{}, nil
	})
}
