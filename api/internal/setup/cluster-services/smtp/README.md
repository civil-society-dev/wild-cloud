# SMTP Configuration Service

This service configures SMTP settings for Wild Cloud applications to send transactional emails.

## Overview

The SMTP service doesn't deploy any Kubernetes resources. Instead, it helps configure global SMTP settings that can be used by Wild Cloud applications like Ghost, Gitea, and others for sending:

- Password reset emails
- User invitation emails  
- Notification emails
- Other transactional emails

## Installation

```bash
./setup/cluster-services/smtp/install.sh
```

## Configuration

The setup script will prompt for:

- **SMTP Host**: Your email provider's SMTP server (e.g., `email-smtp.us-east-2.amazonaws.com` for AWS SES)
- **SMTP Port**: Usually `465` for SSL or `587` for STARTTLS
- **SMTP User**: Username or access key for authentication
- **From Address**: Default sender email address
- **SMTP Password**: Your password, secret key, or API key (entered securely)

## Supported Providers

- **AWS SES**: Use your Access Key ID as user and Secret Access Key as password
- **Gmail/Google Workspace**: Use your email as user and an App Password as password
- **SendGrid**: Use `apikey` as user and your API key as password
- **Mailgun**: Use your Mailgun username and password
- **Other SMTP providers**: Use your standard SMTP credentials

## Applications That Use SMTP

- **Ghost**: User management, password resets, notifications
- **Gitea**: User registration, password resets, notifications
- **OpenProject**: User invitations, notifications
- **Future applications**: Any app that needs to send emails

## Testing

After configuration, test SMTP by:

1. Deploying an application that uses email (like Ghost)
2. Using password reset or user invitation features
3. Checking application logs for SMTP connection issues