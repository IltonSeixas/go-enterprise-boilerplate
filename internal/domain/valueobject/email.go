package valueobject

import (
	"strings"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
)

type Email struct{ value string }

func NewEmail(v string) (Email, error) {
	trimmed := strings.ToLower(strings.TrimSpace(v))
	if !isValidEmail(trimmed) {
		return Email{}, apperror.ErrInvalidEmail
	}
	return Email{value: trimmed}, nil
}

func (e Email) String() string { return e.value }

func isValidEmail(v string) bool {
	if len(v) == 0 || len(v) > 254 {
		return false
	}
	at := strings.LastIndex(v, "@")
	if at < 1 {
		return false
	}
	local := v[:at]
	domain := v[at+1:]
	if len(local) == 0 || len(domain) == 0 {
		return false
	}
	dot := strings.LastIndex(domain, ".")
	return dot > 0 && dot < len(domain)-1
}
