# API Key System Design

## 🔑 API Key Format
```
sk_live_[48_char_random_key_here]
sk_test_[48_char_random_key_here]
│  │    │                                                  
│  │    └── 48-character random key (base62)               
│  └── Environment (live/test)                              
└── Service prefix (sk = secret key)                       
```

## 🏗️ Key Generation Process
1. **Generate**: 64 random bytes → base62 encode → 48 chars
2. **Prefix**: Add `sk_live_` or `sk_test_`
3. **Store**: Hash the full key (SHA-256) + store prefix for display
4. **Return**: Full key once (never stored in plain text)

## 🔒 Authentication Flow
```
Customer Request:
  Headers: Authorization: Bearer sk_live_abc123...
  
API Gateway:
  1. Extract key from header
  2. Hash the key (SHA-256)
  3. Look up hash in api_keys table
  4. Verify: is_active=true, not expired, user active
  5. Check rate limits
  6. Log usage
  7. Forward to core API
```

## 🎯 Rate Limiting Strategy

### Per-User Limits (based on plan)
- **Free**: 100/hour, 1,000/day, 10,000/month
- **Starter**: 1,000/hour, 10,000/day, 100,000/month  
- **Pro**: 5,000/hour, 50,000/day, 500,000/month
- **Enterprise**: 20,000/hour, 200,000/day, 2M/month

### Implementation
- **Redis counters**: `user:{user_id}:hour:{YYYY-MM-DD-HH}` = count
- **Sliding window**: Check last hour/day/month usage
- **Burst allowance**: Allow 10 extra requests in short bursts
- **429 responses**: With headers showing limits and reset times

### Response Headers
```
X-RateLimit-Limit-Hour: 1000
X-RateLimit-Remaining-Hour: 847
X-RateLimit-Reset-Hour: 1694789200
X-RateLimit-Limit-Day: 10000  
X-RateLimit-Remaining-Day: 9153
```

## 🎮 Customer Dashboard Features

### Account Management
- **Profile**: Edit company info, billing details
- **API Keys**: Create, name, rotate, delete keys
- **Usage Analytics**: Charts showing request patterns
- **Plan Management**: Upgrade/downgrade, billing

### Monitoring & Analytics  
- **Real-time usage**: Current hour/day/month counts
- **Usage history**: Daily/weekly/monthly trends
- **Popular categories**: Which types of prompts most used
- **Model preferences**: Which models recommended most
- **Error rates**: Track failed requests and reasons
- **Performance metrics**: Average response times

## 🚨 Security Features

### Key Security
- **Never log full keys**: Only log prefixes in error messages
- **Rotation support**: Generate new key, transition, delete old
- **Expiration**: Optional key expiry dates
- **IP restrictions**: Whitelist allowed IP ranges (enterprise)
- **Permissions**: Scope keys to specific endpoints

### Account Security
- **Email verification**: Required for account activation
- **Password requirements**: 8+ chars, complexity rules
- **Login rate limiting**: Prevent brute force attacks
- **Session management**: JWT tokens with refresh
- **Audit logs**: Track all account changes

## 🔧 Implementation Priority

### Phase 1: Core Authentication
1. Database schema setup
2. User registration/login API
3. API key generation/validation
4. Basic rate limiting

### Phase 2: Dashboard & Management
1. Web dashboard for key management
2. Usage tracking and analytics
3. Plan management
4. Advanced rate limiting

### Phase 3: Enterprise Features
1. Team management
2. IP restrictions
3. Custom quotas
4. Advanced analytics
5. Webhook notifications