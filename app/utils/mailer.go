package utils

import (
	"fmt"
	"net/smtp"
	"os"
)

// SendResetEmail sends a password reset link using SMTP credentials from the environment.
// It uses SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, and MAIL_FROM.
func SendResetEmail(to, token string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("MAIL_FROM")

	if host == "" || port == "" || user == "" || pass == "" || from == "" {
		return fmt.Errorf("SMTP credentials missing in environment")
	}

	// 1. Compose the message
	subject := "Subject: Password Reset Request (InkToChat)\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173" // Fallback
	}
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, token)
	
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>InkToChat: Password Reset Request</h2>
			<p>An administrator has initiated a password reset for your account.</p>
			<p>Please click the link below to set a new password. This link is valid for <b>1 hour</b>.</p>
			<a href="%s" style="padding: 10px 20px; background-color: #4CAF50; color: white; text-decoration: none; border-radius: 5px;">Reset My Password</a>
			<p>If you did not request this, please ignore this email.</p>
			<hr>
			<p><small>Sent from DigitalPlat Social Media Analytics Backend</small></p>
		</body>
		</html>`, resetURL)

	msg := []byte(subject + mime + body)

	// 2. Authenticate
	auth := smtp.PlainAuth("", user, pass, host)

	// 3. Send
	addr := fmt.Sprintf("%s:%s", host, port)
	err := smtp.SendMail(addr, auth, from, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

// SendTestEmail sends a simple test email to verify credentials.
func SendTestEmail(to string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("MAIL_FROM")

	subject := "Subject: SMTP Test Email\n"
	body := "Your SMTP configuration is working correctly!"
	msg := []byte(subject + "\n" + body)

	auth := smtp.PlainAuth("", user, pass, host)
	addr := fmt.Sprintf("%s:%s", host, port)
	
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
