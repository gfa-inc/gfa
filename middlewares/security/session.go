package security

import (
	"errors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

var SessionKey = "user"

var ErrNotFoundSession = errors.New("not found session")

type SessionValidator struct {
}

func (s *SessionValidator) Valid(c *gin.Context) error {
	session := sessions.Default(c)
	v := session.Get(SessionKey)
	if v == nil {
		return ErrNotFoundSession
	}

	return nil
}
