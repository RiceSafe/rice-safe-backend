package email

import "context"

// MockEmailService implements Service for use in tests.
// It captures the last call's arguments so tests can assert on them.
type MockEmailService struct {
	SendPasswordResetFunc func(ctx context.Context, to, code string) error
	LastTo                string
	LastCode              string
}

func (m *MockEmailService) SendPasswordReset(ctx context.Context, to, code string) error {
	m.LastTo = to
	m.LastCode = code
	if m.SendPasswordResetFunc != nil {
		return m.SendPasswordResetFunc(ctx, to, code)
	}
	return nil
}
