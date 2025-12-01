package helpers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
)

type Paginated struct {
	NextToken  *string `json:"nextToken" form:"nextToken"`
	MaxResults *int64  `json:"maxResults" form:"maxResults" validate:"max=1600;min=1"`
}

type Sorted struct {
	SortBy    *string `json:"sortBy" form:"sortBy"`
	SortOrder *string `json:"sortOrder" form:"sortOrder" validate:"oneofci=asc desc"`
}

type HttpError struct {
	Code    int    `json:"code"`
	Details string `json:"details"`
}

func (e HttpError) Error() string {
	return e.Details
}

type QueryHandlerFunc[T any] func(query T, c *gin.Context) (any, error)

func QueryHandler[T any](f QueryHandlerFunc[T]) gin.HandlerFunc {
	return BasicHandler(func(c *gin.Context) (any, error) {
		var query T

		if err := c.ShouldBindQuery(&query); err != nil {
			return nil, HttpError{Code: 400, Details: err.Error()}
		}

		return f(query, c)
	})
}

type BodyHandlerFunc[T any] func(body T, c *gin.Context) (any, error)

func BodyHandler[T any](f BodyHandlerFunc[T]) gin.HandlerFunc {
	return BasicHandler(func(c *gin.Context) (any, error) {
		var body T

		if err := c.ShouldBindBodyWithJSON(&body); err != nil {
			return nil, HttpError{Code: http.StatusBadRequest, Details: err.Error()}
		}

		return f(body, c)
	})
}

type BasicHandlerFunc func(c *gin.Context) (any, error)

func BasicHandler(f BasicHandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := f(c)

		var httpError *HttpError
		var teaError *tea.SDKError

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, Details("no rows found"))
				return
			}

			if errors.As(err, &httpError) {
				c.JSON(httpError.Code, Details(err.Error()))
				return
			}

			if errors.As(err, &teaError) {
				c.JSON(*teaError.StatusCode, Details(fmt.Sprintf("sdk:%s", *teaError.Code)))
				return
			}

			if errors.Is(err, context.DeadlineExceeded) {
				c.JSON(http.StatusRequestTimeout, Details("request timeout"))
				return
			}

			c.JSON(http.StatusInternalServerError, Details(err.Error()))
			return
		}

		c.JSON(http.StatusOK, res)
	}
}
