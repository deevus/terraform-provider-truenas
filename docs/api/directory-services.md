# Directory Services API

Active Directory, LDAP, and identity provider integration.

## Directory Services Status

### directoryservices.status
Get directory services status.
```bash
midclt call directoryservices.status
```

Returns:
- `type` - ACTIVEDIRECTORY, LDAP, null
- `status` - DISABLED, JOINING, HEALTHY, FAULTED
- `status_msg` - Status message

### directoryservices.config
Get directory services configuration.
```bash
midclt call directoryservices.config
```

### directoryservices.update
Update directory services configuration.
```bash
midclt call directoryservices.update '{
  "domainname": "example.com",
  "binddn": "CN=binduser,CN=Users,DC=example,DC=com",
  "bindpw": "password",
  "basedn": "DC=example,DC=com"
}'
```

### directoryservices.leave
Leave domain.
```bash
midclt call directoryservices.leave '{"username": "admin", "password": "password"}'
```

### directoryservices.cache_refresh
Refresh directory cache.
```bash
midclt call directoryservices.cache_refresh
```

### directoryservices.sync_keytab
Sync Kerberos keytab.
```bash
midclt call directoryservices.sync_keytab
```

### directoryservices.certificate_choices
Get available certificates for directory services.
```bash
midclt call directoryservices.certificate_choices
```

## Kerberos Realms

### kerberos.realm.query
Query Kerberos realms.
```bash
midclt call kerberos.realm.query
```

### kerberos.realm.create
Create a Kerberos realm.
```bash
midclt call kerberos.realm.create '{
  "realm": "EXAMPLE.COM",
  "kdc": ["kdc1.example.com", "kdc2.example.com"],
  "admin_server": ["admin.example.com"],
  "kpasswd_server": ["kpasswd.example.com"]
}'
```

### kerberos.realm.update
Update a Kerberos realm.
```bash
midclt call kerberos.realm.update <realm_id> '{
  "kdc": ["newkdc.example.com"]
}'
```

### kerberos.realm.delete
Delete a Kerberos realm.
```bash
midclt call kerberos.realm.delete <realm_id>
```

## Kerberos Keytabs

### kerberos.keytab.query
Query Kerberos keytabs.
```bash
midclt call kerberos.keytab.query
```

### kerberos.keytab.create
Create a Kerberos keytab.
```bash
midclt call kerberos.keytab.create '{
  "name": "nfs-keytab",
  "file": "<base64-encoded-keytab>"
}'
```

### kerberos.keytab.update
Update a Kerberos keytab.
```bash
midclt call kerberos.keytab.update <keytab_id> '{
  "file": "<base64-encoded-keytab>"
}'
```

### kerberos.keytab.delete
Delete a Kerberos keytab.
```bash
midclt call kerberos.keytab.delete <keytab_id>
```

### kerberos.keytab.kerberos_principal_choices
Get available Kerberos principals.
```bash
midclt call kerberos.keytab.kerberos_principal_choices
```

## Active Directory Configuration

Active Directory is configured through `directoryservices.update`:

```bash
midclt call directoryservices.update '{
  "domainname": "ad.example.com",
  "binddn": "Administrator",
  "bindpw": "password",
  "enable": true,
  "verbose_logging": false,
  "use_default_domain": true,
  "allow_trusted_doms": false,
  "allow_dns_updates": true,
  "disable_freenas_cache": false,
  "restrict_pam": false,
  "site": null,
  "kerberos_realm": <realm_id>,
  "kerberos_principal": "<principal>",
  "timeout": 60,
  "dns_timeout": 10,
  "nss_info": "RFC2307",
  "createcomputer": "Computers"
}'
```

## LDAP Configuration

LDAP is also configured through `directoryservices.update`:

```bash
midclt call directoryservices.update '{
  "hostname": ["ldap.example.com"],
  "basedn": "dc=example,dc=com",
  "binddn": "cn=admin,dc=example,dc=com",
  "bindpw": "password",
  "enable": true,
  "ssl": "ON",
  "certificate": <cert_id>,
  "validate_certificates": true,
  "kerberos_realm": null,
  "kerberos_principal": null,
  "timeout": 30,
  "dns_timeout": 10,
  "schema": "RFC2307",
  "auxiliary_parameters": ""
}'
```

## NSS Info Options

| Option | Description |
|--------|-------------|
| `RFC2307` | RFC 2307 schema |
| `RFC2307BIS` | RFC 2307bis schema |
| `SFU` | Services for Unix |
| `SFU20` | Services for Unix 2.0 |

## SSL Options

| Option | Description |
|--------|-------------|
| `OFF` | No SSL |
| `ON` | SSL enabled |
| `START_TLS` | STARTTLS |
