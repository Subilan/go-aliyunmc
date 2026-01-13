package events

import "github.com/gin-gonic/gin"

func Error(details string) *Event {
	return Stateless(gin.H{"details": details}, TypeError, false)
}
