package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v3"
)

// ResendEmailService implements Service using the Resend API.
type ResendEmailService struct {
	client    *resend.Client
	fromEmail string
}

// NewResendService creates a new ResendEmailService.
func NewResendService(apiKey, fromEmail string) Service {
	return &ResendEmailService{
		client:    resend.NewClient(apiKey),
		fromEmail: fromEmail,
	}
}

// SendPasswordReset sends a 6-digit OTP code to the given email address.
func (s *ResendEmailService) SendPasswordReset(ctx context.Context, to, code string) error {
	params := &resend.SendEmailRequest{
		From:    s.fromEmail,
		To:      []string{to},
		Subject: "RiceSafe - Password Reset Code",
		Html: fmt.Sprintf(`
			<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
				<h2>Password Reset Request</h2>
				<p>Use the code below to reset your RiceSafe password. It expires in <strong>15 minutes</strong>.</p>
				<div style="font-size: 32px; font-weight: bold; letter-spacing: 8px; padding: 16px;
				            background: #f4f4f4; border-radius: 8px; text-align: center;">
					%s
				</div>
				<p style="color: #888; font-size: 12px; margin-top: 16px;">
					If you did not request this, you can safely ignore this email.
				</p>
			</div>
		`, code),
	}

	_, err := s.client.Emails.Send(params)
	return err
}
