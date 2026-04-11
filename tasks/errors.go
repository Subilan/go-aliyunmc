package tasks

import "errors"

var ErrBadArguments = errors.New("参数不足或不合要求")
var ErrTaskTypeNotFound = errors.New("找不到该任务类型")