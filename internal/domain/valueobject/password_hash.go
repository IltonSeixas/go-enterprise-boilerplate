package valueobject

// PasswordHash is an opaque wrapper around an Argon2id PHC string.
// It can only be constructed by infrastructure — never by domain logic directly.
type PasswordHash struct{ phc string }

func NewPasswordHashFromPHC(phc string) PasswordHash {
	return PasswordHash{phc: phc}
}

func (p PasswordHash) PHC() string { return p.phc }

// String never exposes the hash value to prevent accidental logging.
func (p PasswordHash) String() string { return "[REDACTED]" }
