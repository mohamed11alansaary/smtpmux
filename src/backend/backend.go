package backend

import (
	"github.com/emersion/go-smtp"
	"github.com/goyal-aman/mailmux/src/session"
	"github.com/goyal-aman/mailmux/src/types"
)

// Backend implements smtp.Backend
type Backend struct {
	cfg types.Config
}

func NewBackend(cfg types.Config) *Backend {
	return &Backend{cfg: cfg}
}

func (be *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return session.NewSession(be.cfg), nil
}
