package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
)

func sendInvoiceNotification(inv *Invoice) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_APP_PASSWORD")
	from := os.Getenv("SMTP_FROM")
	to := os.Getenv("NOTIFICATION_EMAIL")
	baseURL := os.Getenv("APP_BASE_URL")

	if host == "" || port == "" || user == "" || pass == "" || from == "" || to == "" {
		log.Printf("email: SMTP env vars missing, skipping notification for invoice %d", inv.ID)
		return nil
	}

	link := fmt.Sprintf("%s/api/invoices/%d/open?template=default_invoice.html",
		strings.TrimRight(baseURL, "/"), inv.ID)

	subject := fmt.Sprintf("Nova fatura gerada: %s — %s", inv.Identification(), inv.Client.Name)
	body := fmt.Sprintf(
		"Uma nova fatura foi gerada automaticamente.\r\n\r\n"+
			"Cliente: %s\r\n"+
			"Identificação: %s\r\n"+
			"Total: R$ %.2f\r\n"+
			"Vencimento: %s\r\n\r\n"+
			"Abrir: %s\r\n",
		inv.Client.Name,
		inv.Identification(),
		inv.Total(),
		inv.DueDate.Format("2006-01-02"),
		link,
	)

	msg := []byte(
		"From: " + from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			body,
	)

	addr := net.JoinHostPort(host, port)
	auth := smtp.PlainAuth("", user, pass, host)

	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer c.Close()

	if err := c.Hello("localhost"); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}
	if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
		return fmt.Errorf("smtp starttls: %w", err)
	}
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return c.Quit()
}
