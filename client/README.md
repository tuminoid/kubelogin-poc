# OAuth2 Token Lifecycle and Testing

This document explains OAuth2 token mechanics, test scenario configuration,
and production recommendations for Dex OIDC provider.

## Token Types and Relationships

### ID Token (JWT)

**Purpose**: Primary token containing user identity and authorization claims

- **Format**: JSON Web Token (JWT)
- **Contents**: User identity, groups, permissions, authorization claims
- **Lifetime**: Short (15min - 1h)
- **Usage**: Both identity verification AND API authorization in Kubernetes
- **Note**: In Dex/Kubernetes OIDC, this serves as both ID and access token

### Refresh Token (Opaque)

**Purpose**: Obtaining new ID tokens without re-authentication

- **Format**: Opaque string (implementation-specific)
- **Contents**: Encrypted session reference
- **Lifetime**: Long (days/weeks)
- **Usage**: Single-use token for acquiring new tokens

## Token Hierarchy and Dependencies

```text
User Authentication
         ↓
    [Refresh Token] ← Long-lived, single-use
         ↓
    Token Refresh Request
         ↓
   ┌─────────────┬─────────────┐
   ↓             ↓             ↓
   [ID Token]  [ID Token]  [New Refresh Token]
   (identity)  (as access)     ↓
       ↓           ↓      Next refresh
  User Info   API calls
```

### Token Rotation

Dex implements refresh token rotation pattern:

1. Refresh token is **single-use only**
2. Each refresh returns a **new refresh token**
3. Old refresh token is **immediately invalidated**
4. Attempting to reuse old token → `invalid_request` error

## Dex Token Expiry Configuration

### Core Settings

```yaml
expiry:
  idTokens: "30m"             # JWT lifetime for API calls
  authRequests: "5m"          # Login flow timeout
  deviceRequests: "5m"        # Device code validity
  refreshTokens:
    validIfNotUsedFor: "72h"  # Idle timeout
    absoluteLifetime: "720h"  # Maximum age
    reuseInterval: "3s"       # Grace period for rotation
```

### Setting Purposes

- **idTokens**: Controls API session length
- **authRequests**: Time to complete login
- **deviceRequests**: Device code expiration (device-code flow)
- **validIfNotUsedFor**: Logout inactive users
- **absoluteLifetime**: Force periodic re-authentication
- **reuseInterval**: Prevents race conditions during rotation

### Test vs Production

Test configuration uses short values for rapid iteration:

- `idTokens: "30s"` → Forces frequent refresh testing
- `validIfNotUsedFor: "5m"` → Quick idle timeout
- `absoluteLifetime: "15m"` → Short session for testing

Production should use longer, security-appropriate values.

## References and Documentation

- <https://dexidp.io/docs/configuration/tokens/>
