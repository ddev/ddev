package types

import "time"

type MessagesStorage interface {
	LastUpdate() time.Time
	Push(messages *Messages) error
	Pull() (Messages, error)
}
