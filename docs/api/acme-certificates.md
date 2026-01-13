# ACME & Certificate API

SSL/TLS certificate management and ACME automation.

## Certificate Management

### certificate.query
Query certificates.
```bash
midclt call certificate.query
midclt call certificate.query '[[["name", "=", "truenas_default"]]]'
```

Returns:
- `id` - Certificate ID
- `name` - Certificate name
- `certificate` - PEM certificate
- `privatekey` - PEM private key
- `CSR` - Certificate signing request
- `cert_type` - CERTIFICATE, CA, CSR
- `revoked` - Revoked flag
- `can_be_revoked` - Can be revoked
- `internal` - Internally generated
- `CA_type_existing` - Existing CA type
- `CA_type_internal` - Internal CA type
- `CA_type_intermediate` - Intermediate CA type
- `cert_type_existing` - Existing cert type
- `cert_type_internal` - Internal cert type
- `cert_type_CSR` - CSR cert type
- `signedby` - Signing CA
- `root_path` - Certificate file path
- `certificate_path` - Certificate file path
- `privatekey_path` - Private key file path
- `csr_path` - CSR file path
- `chain` - Full certificate chain
- `issuer` - Issuer info
- `chain_list` - Chain as list
- `country` - Country
- `state` - State
- `city` - City
- `organization` - Organization
- `organizational_unit` - OU
- `san` - Subject alternative names
- `email` - Email
- `DN` - Distinguished name
- `subject_name_hash` - Subject hash
- `digest_algorithm` - Digest algorithm
- `lifetime` - Lifetime in days
- `from` - Valid from
- `until` - Valid until
- `serial` - Serial number
- `common` - Common name
- `fingerprint` - Certificate fingerprint
- `expired` - Expired flag
- `parsed` - Parsed certificate
- `extensions` - X.509 extensions

### certificate.create
Create a certificate.

Import existing certificate:
```bash
midclt call certificate.create '{
  "name": "imported-cert",
  "create_type": "CERTIFICATE_CREATE_IMPORTED",
  "certificate": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
  "privatekey": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----"
}'
```

Create self-signed certificate:
```bash
midclt call certificate.create '{
  "name": "self-signed",
  "create_type": "CERTIFICATE_CREATE_INTERNAL",
  "signedby": <ca_id>,
  "key_type": "RSA",
  "key_length": 2048,
  "digest_algorithm": "SHA256",
  "lifetime": 3650,
  "country": "US",
  "state": "California",
  "city": "San Francisco",
  "organization": "My Org",
  "organizational_unit": "IT",
  "email": "admin@example.com",
  "common": "truenas.example.com",
  "san": ["DNS:truenas.example.com", "IP:192.168.1.10"]
}'
```

Create CSR:
```bash
midclt call certificate.create '{
  "name": "csr-cert",
  "create_type": "CERTIFICATE_CREATE_CSR",
  "key_type": "RSA",
  "key_length": 2048,
  "digest_algorithm": "SHA256",
  "country": "US",
  "state": "California",
  "city": "San Francisco",
  "organization": "My Org",
  "common": "truenas.example.com",
  "san": ["DNS:truenas.example.com"]
}'
```

ACME certificate:
```bash
midclt call certificate.create '{
  "name": "acme-cert",
  "create_type": "CERTIFICATE_CREATE_ACME",
  "acme_directory_uri": "https://acme-v02.api.letsencrypt.org/directory",
  "dns_mapping": {
    "example.com": <acme_dns_auth_id>
  },
  "tos": true,
  "domains": ["truenas.example.com"]
}'
```

### certificate.update
Update a certificate.
```bash
midclt call certificate.update <cert_id> '{"name": "new-name"}'
```

### certificate.delete
Delete a certificate.
```bash
midclt call certificate.delete <cert_id>
```

### certificate.country_choices
Get country choices.
```bash
midclt call certificate.country_choices
```

### certificate.ec_curve_choices
Get EC curve choices.
```bash
midclt call certificate.ec_curve_choices
```

### certificate.extended_key_usage_choices
Get extended key usage choices.
```bash
midclt call certificate.extended_key_usage_choices
```

### certificate.acme_server_choices
Get ACME server choices.
```bash
midclt call certificate.acme_server_choices
```

## ACME DNS Authenticators

### acme.dns.authenticator.query
Query DNS authenticators.
```bash
midclt call acme.dns.authenticator.query
```

### acme.dns.authenticator.authenticator_schemas
Get authenticator schemas.
```bash
midclt call acme.dns.authenticator.authenticator_schemas
```

### acme.dns.authenticator.create
Create a DNS authenticator.

Cloudflare:
```bash
midclt call acme.dns.authenticator.create '{
  "name": "cloudflare",
  "authenticator": "cloudflare",
  "attributes": {
    "cloudflare_email": "admin@example.com",
    "api_key": "global_api_key",
    "api_token": ""
  }
}'
```

Or with API token:
```bash
midclt call acme.dns.authenticator.create '{
  "name": "cloudflare-token",
  "authenticator": "cloudflare",
  "attributes": {
    "cloudflare_email": "",
    "api_key": "",
    "api_token": "api_token_here"
  }
}'
```

Route53:
```bash
midclt call acme.dns.authenticator.create '{
  "name": "route53",
  "authenticator": "route53",
  "attributes": {
    "access_key_id": "AKIAXXXXXXXX",
    "secret_access_key": "secretkey"
  }
}'
```

### acme.dns.authenticator.update
Update a DNS authenticator.
```bash
midclt call acme.dns.authenticator.update <auth_id> '{"attributes": {"api_token": "new_token"}}'
```

### acme.dns.authenticator.delete
Delete a DNS authenticator.
```bash
midclt call acme.dns.authenticator.delete <auth_id>
```

## WebUI Crypto Helpers

### webui.crypto.csr_profiles
Get CSR profiles.
```bash
midclt call webui.crypto.csr_profiles
```

### webui.crypto.get_certificate_domain_names
Get domain names from certificate.
```bash
midclt call webui.crypto.get_certificate_domain_names <cert_id>
```

## Certificate Types

| Create Type | Description |
|-------------|-------------|
| `CERTIFICATE_CREATE_INTERNAL` | Self-signed certificate |
| `CERTIFICATE_CREATE_IMPORTED` | Import existing certificate |
| `CERTIFICATE_CREATE_CSR` | Create CSR for external CA |
| `CERTIFICATE_CREATE_ACME` | ACME/Let's Encrypt certificate |

## Key Types

| Type | Description |
|------|-------------|
| `RSA` | RSA key |
| `EC` | Elliptic curve key |

## Digest Algorithms

| Algorithm | Description |
|-----------|-------------|
| `SHA1` | SHA-1 (deprecated) |
| `SHA224` | SHA-224 |
| `SHA256` | SHA-256 (recommended) |
| `SHA384` | SHA-384 |
| `SHA512` | SHA-512 |

## Common ACME Servers

| Server | URI |
|--------|-----|
| Let's Encrypt Production | `https://acme-v02.api.letsencrypt.org/directory` |
| Let's Encrypt Staging | `https://acme-staging-v02.api.letsencrypt.org/directory` |

## Complete ACME Setup Example

```bash
# 1. Create DNS authenticator (Cloudflare)
midclt call acme.dns.authenticator.create '{
  "name": "cloudflare-dns",
  "authenticator": "cloudflare",
  "attributes": {"api_token": "your_cloudflare_token"}
}'

# 2. Create ACME certificate
midclt call certificate.create '{
  "name": "letsencrypt-cert",
  "create_type": "CERTIFICATE_CREATE_ACME",
  "acme_directory_uri": "https://acme-v02.api.letsencrypt.org/directory",
  "dns_mapping": {"example.com": 1},
  "tos": true,
  "domains": ["truenas.example.com"]
}'

# 3. Set as UI certificate
midclt call system.general.update '{"ui_certificate": <new_cert_id>}'

# 4. Restart UI
midclt call system.general.ui_restart
```
