package models

import (
	"time"

	"github.com/aarondl/opt/null"
)

type User struct {
	ID       int              `json:"id"`
	Username string           `json:"username"`
	Email    null.Val[string] `json:"email"`

	Shop   Company           `json:"company"`
	Extras map[string]string `json:"extras,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}
