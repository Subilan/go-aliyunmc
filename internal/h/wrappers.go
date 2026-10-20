package h

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// BasicHandler 表示一个基本的路由处理逻辑
type BasicHandler func(c *gin.Context) (any, error)

// BodyHandler 表示一个专门处理请求体的路由处理逻辑
type BodyHandler[T any] func(data T, c *gin.Context) (any, error)

// QueryHandler 表示一个专门处理查询参数的路由处理逻辑
type QueryHandler[T any] func(data T, c *gin.Context) (any, error)

// G 用于将一个 BasicHandler 包装成 gin.HandlerFunc，它是最基础的包装函数，同时也是所有其他包装函数的基础。
// G 中包含了对于返回的错误信息的进一步检查，将其对应到适当的HTTP状态码上。
func G(handler BasicHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := handler(c)

		var httpErr *httpError
		if err != nil {
			// 依次检查错误类型
			// 如果最终匹配不到任何错误类型，返回 500

			// 如果错误是 httpError 类型，返回对应的HTTP状态码
			if errors.As(err, &httpErr) {
				c.JSON(httpErr.Code, Details(httpErr))
				return
			}

			// 如果错误是 gorm.ErrDuplicatedKey 类型，返回 409
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				c.JSON(http.StatusConflict, Details(err))
				return
			}

			// 如果错误是 gorm.ErrRecordNotFound 类型，返回 404
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, Details(err))
				return
			}

			c.JSON(http.StatusInternalServerError, Details(err))
			return
		}
		c.JSON(http.StatusOK, Data(data))
	}
}

// B 用于将一个 BodyHandler 包装成 gin.HandlerFunc
func B[T any](h BodyHandler[T]) gin.HandlerFunc {
	return G(func(c *gin.Context) (any, error) {
		var body T
		if err := c.ShouldBindJSON(&body); err != nil {
			return nil, err
		}
		return h(body, c)
	})
}

// Q 用于将一个 QueryHandler 包装成 gin.HandlerFunc
func Q[T any](h QueryHandler[T]) gin.HandlerFunc {
	return G(func(c *gin.Context) (any, error) {
		var query T
		if err := c.ShouldBindQuery(&query); err != nil {
			return nil, err
		}
		return h(query, c)
	})
}

// V 将一个简单的返回一个值的函数包装为始终返回 200 的 gin.HandlerFunc
func V[T any](value func() T) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, Data(value()))
	}
}

// VE 将一个返回一个值和一个错误的函数包装为 gin.HandlerFunc
func VE[T any](value func() (T, error)) gin.HandlerFunc {
	return G(func(ctx *gin.Context) (any, error) {
		return value()
	})
}

// VB 将一个返回一个值和一个布尔值的函数包装为 gin.HandlerFunc，如果布尔值为 false，则返回 404
func VB[T any](value func() (T, bool)) gin.HandlerFunc {
	return G(func(ctx *gin.Context) (any, error) {
		v, ok := value()
		if !ok {
			return nil, HttpError(http.StatusNotFound, "暂时无法获取该数据")
		}
		return v, nil
	})
}
