package oss_routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/clients"
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/gin-gonic/gin"
)

type ListObjectsQuery struct {
	Target     string `form:"target" binding:"required,oneof=backups"`
	TrimPrefix bool   `form:"trimPrefix"`
}

type ListObjectsResponseItem struct {
	Name         string     `json:"name"`
	Size         int64      `json:"size"`
	LastModified *time.Time `json:"lastModified"`
}

func ListObjects() gin.HandlerFunc {
	return helpers.QueryHandler[ListObjectsQuery](func(query ListObjectsQuery, c *gin.Context) (any, error) {
		var prefix string
		switch query.Target {
		case "backups":
			prefix = "backups"
		default:
			return nil, fmt.Errorf("target %s not supported", query.Target)
		}

		res, err := clients.OssClient.ListObjectsV2(c.Request.Context(), &oss.ListObjectsV2Request{
			Bucket: tea.String(config.Cfg.Deploy.BucketName()),
			Prefix: tea.String(prefix),
		})

		if err != nil {
			return nil, err
		}

		var result = make([]ListObjectsResponseItem, 0, 5)

		for _, item := range res.Contents {
			name := *item.Key

			if query.TrimPrefix {
				name = strings.TrimPrefix(name, prefix+"/")
			}

			resultItem := ListObjectsResponseItem{
				Name:         name,
				Size:         item.Size,
				LastModified: item.LastModified,
			}
			result = append(result, resultItem)
		}

		return helpers.Data(result), nil
	})
}
