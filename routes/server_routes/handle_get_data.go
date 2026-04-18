package server_routes

import (
	"encoding/json"
	"go-aliyunmc/h"
	"go-aliyunmc/monitors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
)

var supportedFileSuffix = [...]string{"json", "yml", "yaml"}
var supportedFileHandlers map[string]func([]byte) (map[string]any, error)

func init() {
	jsonHandler := func(data []byte) (map[string]any, error) {
		var result map[string]any
		err := json.Unmarshal(data, &result)
		return result, err
	}

	yamlHandler := func(data []byte) (map[string]any, error) {
		var result map[string]any
		err := yaml.Unmarshal(data, &result)
		return result, err
	}

	supportedFileHandlers = map[string]func([]byte) (map[string]any, error){
		"json": jsonHandler,
		"yaml": yamlHandler,
		"yml":  yamlHandler,
	}
}

func HandleGetData(c *gin.Context) (any, error) {
	name := c.Query("name")

	if name == "" {
		return nil, h.HttpError(http.StatusBadRequest, "请提供文件名")
	}

	for _, file := range monitors.FileSyncC.Files {
		fileSuffix := ""
		for _, suffix := range supportedFileSuffix {
			if strings.HasSuffix(file.LocalPath, "."+suffix) {
				fileSuffix = suffix
				break
			}
		}

		if fileSuffix == "" {
			continue
		}

		if file.Id != "" && file.Id == name || file.LocalPath == name {
			filePath := monitors.FileSyncC.LocalCacheRoot + file.LocalPath
			content, err := os.ReadFile(filePath)

			if err != nil {
				return nil, err
			}

			data, err := supportedFileHandlers[fileSuffix](content)

			if err != nil {
				return nil, err
			}

			return data, nil
		}
	}

	return nil, h.HttpError(http.StatusNotFound, "未找到指定文件")
}
