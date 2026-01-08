package helpers

import (
	"context"
	"log"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/helpers/stream"
	"github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

// DeleteInstance 完成一次删除实例的业务流程，包括调用API进行删除、更新数据库以及向用户广播删除行为。
func DeleteInstance(ctx context.Context, instanceId string, force bool) error {
	deleteInstanceRequest := &client.DeleteInstanceRequest{
		InstanceId: &instanceId,
		Force:      &force,
		ForceStop:  &force,
	}

	_, err := globals.EcsClient.DeleteInstanceWithContext(ctx, deleteInstanceRequest, &dara.RuntimeOptions{})

	if err != nil {
		return err
	}

	_, err = db.Pool.ExecContext(ctx, "UPDATE instances SET deleted_at = CURRENT_TIMESTAMP WHERE instance_id = ?", instanceId)

	if err != nil {
		return err
	}

	// 将实例删除广播给所有用户
	event, err := store.BuildInstanceEvent(store.InstanceEventNotify, store.InstanceNotificationDeleted)

	if err != nil {
		log.Println("cannot build event:", err)
	}

	err = stream.BroadcastAndSave(event)

	if err != nil {
		log.Println("cannot broadcast and save event:", err)
	}

	return nil
}
