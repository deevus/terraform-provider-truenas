# Mail API

Email configuration and sending.

## Mail Configuration

### mail.config
Get mail configuration.
```bash
midclt call mail.config
```

Returns:
- `id` - Config ID
- `fromemail` - From email address
- `fromname` - From display name
- `outgoingserver` - SMTP server
- `port` - SMTP port
- `security` - PLAIN, SSL, TLS
- `smtp` - SMTP auth enabled
- `user` - SMTP username
- `oauth` - OAuth settings

### mail.update
Update mail configuration.

Basic SMTP:
```bash
midclt call mail.update '{
  "fromemail": "truenas@example.com",
  "fromname": "TrueNAS Server",
  "outgoingserver": "smtp.example.com",
  "port": 587,
  "security": "TLS",
  "smtp": true,
  "user": "truenas@example.com",
  "pass": "password"
}'
```

Gmail with app password:
```bash
midclt call mail.update '{
  "fromemail": "your-email@gmail.com",
  "fromname": "TrueNAS",
  "outgoingserver": "smtp.gmail.com",
  "port": 587,
  "security": "TLS",
  "smtp": true,
  "user": "your-email@gmail.com",
  "pass": "app-specific-password"
}'
```

Office 365:
```bash
midclt call mail.update '{
  "fromemail": "truenas@yourdomain.com",
  "fromname": "TrueNAS",
  "outgoingserver": "smtp.office365.com",
  "port": 587,
  "security": "TLS",
  "smtp": true,
  "user": "truenas@yourdomain.com",
  "pass": "password"
}'
```

With OAuth (Gmail):
```bash
midclt call mail.update '{
  "fromemail": "your-email@gmail.com",
  "fromname": "TrueNAS",
  "outgoingserver": "smtp.gmail.com",
  "port": 587,
  "security": "TLS",
  "smtp": true,
  "oauth": {
    "client_id": "your-client-id",
    "client_secret": "your-client-secret",
    "refresh_token": "your-refresh-token"
  }
}'
```

### mail.local_administrator_email
Get local administrator email.
```bash
midclt call mail.local_administrator_email
```

### mail.send
Send a test email.
```bash
midclt call mail.send '{
  "subject": "Test Email",
  "text": "This is a test email from TrueNAS.",
  "to": ["admin@example.com"]
}'
```

With HTML:
```bash
midclt call mail.send '{
  "subject": "Test Email",
  "text": "Plain text version",
  "html": "<h1>HTML Version</h1><p>This is a test.</p>",
  "to": ["admin@example.com"]
}'
```

With attachments:
```bash
midclt call mail.send '{
  "subject": "Report",
  "text": "Please find the attached report.",
  "to": ["admin@example.com"],
  "attachments": [
    {
      "filename": "report.txt",
      "content": "UmVwb3J0IGNvbnRlbnQ="
    }
  ]
}'
```

## Security Options

| Security | Description | Default Port |
|----------|-------------|--------------|
| `PLAIN` | No encryption | 25 |
| `SSL` | SSL/TLS (implicit) | 465 |
| `TLS` | STARTTLS | 587 |

## Common SMTP Servers

| Provider | Server | Port | Security |
|----------|--------|------|----------|
| Gmail | smtp.gmail.com | 587 | TLS |
| Office 365 | smtp.office365.com | 587 | TLS |
| Outlook.com | smtp-mail.outlook.com | 587 | TLS |
| Yahoo | smtp.mail.yahoo.com | 587 | TLS |
| SendGrid | smtp.sendgrid.net | 587 | TLS |
| Mailgun | smtp.mailgun.org | 587 | TLS |
| Amazon SES | email-smtp.region.amazonaws.com | 587 | TLS |

## Troubleshooting

Test email delivery:
```bash
# Check mail config
midclt call mail.config

# Send test email
midclt call mail.send '{"subject": "Test", "text": "Test", "to": ["test@example.com"]}'
```

Common issues:
- **Gmail**: Requires app-specific password (2FA) or less secure apps
- **Office 365**: May require admin consent for SMTP AUTH
- **Firewall**: Ensure outbound port 587/465 is open
