package helpers

// ErrorResp 表示错误发生时返回的数据结构
type ErrorResp struct {
	// Details 为错误信息
	Details string `json:"details"`
}

// DataResp 表示一个正常返回的数据结构
type DataResp[T any] struct {
	// Data 为该结构中的数据
	Data T `json:"data"`
}

func Details(str string) ErrorResp {
	return ErrorResp{str}
}

func Data[T any](d T) DataResp[T] {
	return DataResp[T]{d}
}
