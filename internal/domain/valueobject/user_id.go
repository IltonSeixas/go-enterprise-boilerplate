package valueobject

import "github.com/google/uuid"

type UserID struct{ value uuid.UUID }

func NewUserID() UserID {
	return UserID{value: uuid.New()}
}

func UserIDFromUUID(id uuid.UUID) UserID {
	return UserID{value: id}
}

func (u UserID) UUID() uuid.UUID  { return u.value }
func (u UserID) String() string   { return u.value.String() }
func (u UserID) IsZero() bool     { return u.value == uuid.Nil }
