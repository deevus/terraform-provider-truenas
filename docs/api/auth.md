# Authentication API

API key management and authentication operations.

## API Keys

### api_key.query
Query API keys.
```bash
midclt call api_key.query
midclt call api_key.query '[[["name", "=", "terraform"]]]'
```

Returns:
- `id` - API key ID
- `name` - Key name
- `key` - Key value (only on create)
- `created_at` - Creation timestamp
- `expires_at` - Expiration timestamp

### api_key.create
Create an API key.
```bash
midclt call api_key.create '{"name": "terraform", "allowlist": []}'
```

### api_key.update
Update an API key.
```bash
midclt call api_key.update <key_id> '{"name": "new-name"}'
```

### api_key.delete
Delete an API key.
```bash
midclt call api_key.delete <key_id>
```

## Authentication

### auth.login_ex
Login with extended authentication.
```bash
midclt call auth.login_ex '{"mechanism": "PASSWORD_PLAIN", "username": "admin", "password": "pass"}'
```

### auth.login_ex_continue
Continue authentication (2FA).
```bash
midclt call auth.login_ex_continue '{"otp_token": "123456"}'
```

### auth.logout
Logout current session.
```bash
midclt call auth.logout
```

### auth.me
Get current user info.
```bash
midclt call auth.me
```

### auth.generate_token
Generate authentication token.
```bash
midclt call auth.generate_token
midclt call auth.generate_token '{"ttl": 3600, "attrs": {"key": "value"}}'
```

### auth.generate_onetime_password
Generate one-time password.
```bash
midclt call auth.generate_onetime_password '{"username": "admin", "ttl": 300}'
```

### auth.mechanism_choices
Get available authentication mechanisms.
```bash
midclt call auth.mechanism_choices
```

### auth.sessions
List active sessions.
```bash
midclt call auth.sessions
```

### auth.terminate_session
Terminate a specific session.
```bash
midclt call auth.terminate_session "<session_id>"
```

### auth.terminate_other_sessions
Terminate all other sessions.
```bash
midclt call auth.terminate_other_sessions
```

### auth.set_attribute
Set session attribute.
```bash
midclt call auth.set_attribute "key" "value"
```

## Two-Factor Authentication

### auth.twofactor.config
Get 2FA configuration.
```bash
midclt call auth.twofactor.config
```

### auth.twofactor.update
Update 2FA configuration.
```bash
midclt call auth.twofactor.update '{
  "enabled": true,
  "otp_digits": 6,
  "interval": 30
}'
```

## Privileges

### privilege.query
Query privileges/roles.
```bash
midclt call privilege.query
```

### privilege.create
Create a privilege.
```bash
midclt call privilege.create '{
  "name": "custom-admin",
  "local_groups": [<group_id>],
  "ds_groups": [],
  "allowlist": [{"method": "CALL", "resource": "*"}],
  "web_shell": true
}'
```

### privilege.update
Update a privilege.
```bash
midclt call privilege.update <privilege_id> '{"web_shell": false}'
```

### privilege.delete
Delete a privilege.
```bash
midclt call privilege.delete <privilege_id>
```

### privilege.roles
Get available roles.
```bash
midclt call privilege.roles
```
