package email

import "context"

// Service defines the contract for sending transactional emails.
type Service interface {
	SendPasswordReset(ctx context.Context, to, code string) error
}
