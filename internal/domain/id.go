package domain

import (
	"github.com/google/uuid"
)

func generateID() string {
	return uuid.New().String()
}
