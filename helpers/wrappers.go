package helpers

import (
	"errors"
	"net/http"

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
	return func(c *gin.Context) {
		var query T

		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, Details(err.Error()))
			return
		}

		res, err := f(query, c)

		var httpError *HttpError

		if err != nil {
			if errors.As(err, &httpError) {
				c.JSON(httpError.Code, Details(err.Error()))
				return
			}

			c.JSON(http.StatusInternalServerError, Details(err.Error()))
			return
		}

		c.JSON(http.StatusOK, res)
	}
}

type BodyHandlerFunc[T any] func(body T, c *gin.Context) (any, error)

func BodyHandler[T any](f BodyHandlerFunc[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body T

		if err := c.ShouldBindBodyWithJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, Details(err.Error()))
			return
		}

		res, err := f(body, c)

		var httpError *HttpError

		if err != nil {
			if errors.As(err, &httpError) {
				c.JSON(httpError.Code, Details(err.Error()))
				return
			}

			c.JSON(http.StatusInternalServerError, Details(err.Error()))
			return
		}

		c.JSON(http.StatusOK, res)
	}
}
