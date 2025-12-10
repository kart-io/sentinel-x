# Rate Limit Security Configuration Examples

This document provides practical examples for configuring the secure rate limiting middleware after the IP extraction security fix.

## Quick Start

### 1. Default Configuration (Secure)
```go
// Secure by default - does not trust proxy headers
middleware := middleware.RateLimit()
```

This configuration:
- Uses `RemoteAddr` for IP extraction (cannot be spoofed)
- Ignores X-Forwarded-For and X-Real-IP headers
- Suitable for direct internet exposure

### 2. Behind a Reverse Proxy (Nginx, Apache)

```go
config := middleware.RateLimitConfig{
    Limit:             100,
    Window:            1 * time.Minute,
    TrustProxyHeaders: true,
    TrustedProxies:    []string{
        "127.0.0.1",  // Local proxy
        "::1",        // IPv6 localhost
    },
}

app.Use(middleware.RateLimitWithConfig(config))
```

### 3. Behind AWS Application Load Balancer

```go
config := middleware.RateLimitConfig{
    Limit:             100,
    Window:            1 * time.Minute,
    TrustProxyHeaders: true,
    TrustedProxies:    []string{
        "10.0.0.0/8",     // Your VPC CIDR
        "172.31.0.0/16",  // Default VPC CIDR (adjust to your actual VPC)
    },
}

app.Use(middleware.RateLimitWithConfig(config))
```

### 4. Behind Cloudflare

```go
config := middleware.RateLimitConfig{
    Limit:             100,
    Window:            1 * time.Minute,
    TrustProxyHeaders: true,
    TrustedProxies:    []string{
        // Cloudflare IPv4 ranges (update periodically)
        "173.245.48.0/20",
        "103.21.244.0/22",
        "103.22.200.0/22",
        "103.31.4.0/22",
        "141.101.64.0/18",
        "108.162.192.0/18",
        "190.93.240.0/20",
        "188.114.96.0/20",
        "197.234.240.0/22",
        "198.41.128.0/17",
        "162.158.0.0/15",
        "104.16.0.0/13",
        "104.24.0.0/14",
        "172.64.0.0/13",
        "131.0.72.0/22",
    },
}

app.Use(middleware.RateLimitWithConfig(config))
```

**Note**: Cloudflare IP ranges can change. Get the latest list from:
https://www.cloudflare.com/ips/

### 5. Multiple Proxy Layers

```go
config := middleware.RateLimitConfig{
    Limit:             100,
    Window:            1 * time.Minute,
    TrustProxyHeaders: true,
    TrustedProxies:    []string{
        "127.0.0.1",           // Local Nginx
        "10.0.0.0/8",          // Internal load balancers
        "192.168.0.0/16",      // Private network
    },
}

app.Use(middleware.RateLimitWithConfig(config))
```

## Security Best Practices

### DO:
1. Use the smallest CIDR range that covers your proxies
2. Regularly review and update proxy IP lists
3. Enable `TrustProxyHeaders` only when necessary
4. Monitor logs for "invalid CIDR" warnings
5. Test configuration in staging before production

### DON'T:
1. Don't use `0.0.0.0/0` in trusted proxies (trusts everyone)
2. Don't enable `TrustProxyHeaders` without configuring `TrustedProxies`
3. Don't trust public IP ranges
4. Don't forget IPv6 addresses if your infrastructure uses them

## Testing Your Configuration

### Test 1: Verify Proxy Headers Are Trusted
```bash
# From your proxy IP
curl -H "X-Forwarded-For: 1.2.3.4" https://your-api.com/test

# Check logs - should see IP: 1.2.3.4
```

### Test 2: Verify Untrusted Sources Are Ignored
```bash
# From a non-proxy IP
curl -H "X-Forwarded-For: 1.2.3.4" https://your-api.com/test

# Check logs - should see actual source IP, NOT 1.2.3.4
```

### Test 3: Verify Rate Limiting Works
```bash
# Make 100+ requests quickly
for i in {1..150}; do
  curl https://your-api.com/test
done

# Should see 429 Too Many Requests after limit
```

## Troubleshooting

### Problem: Rate limiting not working correctly

**Check:**
1. Is `TrustProxyHeaders` set to `true`?
2. Is the proxy IP in `TrustedProxies` list?
3. Are the CIDR ranges correct?
4. Is the proxy setting the correct headers?

**Debug:**
```go
config := middleware.RateLimitConfig{
    Limit:  100,
    Window: 1 * time.Minute,
    TrustProxyHeaders: true,
    TrustedProxies: []string{"127.0.0.1"},
    KeyFunc: func(c transport.Context) string {
        ip := middleware.extractClientIP(c, config)
        logger.Infow("extracted IP", "ip", ip)
        return ip
    },
}
```

### Problem: Getting "invalid CIDR" warnings

**Fix:**
Ensure CIDR notation is correct:
- ✅ `10.0.0.0/8`
- ✅ `192.168.1.1` (single IP, no CIDR needed)
- ❌ `10.0.0.0/999` (invalid prefix length)
- ❌ `not-an-ip/24` (invalid IP)

### Problem: All requests show same IP

**Possible causes:**
1. Proxy not sending X-Forwarded-For header
2. Proxy IP not in `TrustedProxies` list
3. `TrustProxyHeaders` is `false`

**Verify proxy configuration:**
```nginx
# Nginx example
location / {
    proxy_pass http://backend;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Real-IP $remote_addr;
}
```

## Advanced Configurations

### Custom Key Function (Rate limit by user + IP)
```go
config := middleware.RateLimitConfig{
    Limit:  100,
    Window: 1 * time.Minute,
    KeyFunc: func(c transport.Context) string {
        // Extract IP securely
        ip := middleware.extractClientIP(c, config)

        // Get user ID from JWT or session
        userID := c.Get("user_id").(string)

        // Combine for unique key
        return fmt.Sprintf("%s:%s", userID, ip)
    },
    TrustProxyHeaders: true,
    TrustedProxies:    []string{"127.0.0.1"},
}
```

### Different Limits for Different Paths
```go
// Strict limits for authentication
authLimiter := middleware.RateLimitWithConfig(middleware.RateLimitConfig{
    Limit:  10,
    Window: 1 * time.Minute,
    TrustProxyHeaders: true,
    TrustedProxies:    []string{"127.0.0.1"},
})

// Relaxed limits for static content
apiLimiter := middleware.RateLimitWithConfig(middleware.RateLimitConfig{
    Limit:  1000,
    Window: 1 * time.Minute,
    TrustProxyHeaders: true,
    TrustedProxies:    []string{"127.0.0.1"},
})

app.Use("/auth", authLimiter)
app.Use("/api", apiLimiter)
```

### Skip Paths (Health checks, metrics)
```go
config := middleware.RateLimitConfig{
    Limit:  100,
    Window: 1 * time.Minute,
    SkipPaths: []string{
        "/health",
        "/metrics",
        "/readiness",
    },
    TrustProxyHeaders: true,
    TrustedProxies:    []string{"127.0.0.1"},
}
```

## Migration Checklist

- [ ] Identify all proxies/load balancers in front of application
- [ ] Collect IP addresses or CIDR ranges of proxies
- [ ] Test configuration in staging environment
- [ ] Enable `TrustProxyHeaders` and configure `TrustedProxies`
- [ ] Verify rate limiting works as expected
- [ ] Monitor logs for warnings or issues
- [ ] Document configuration for team
- [ ] Set up alerts for configuration changes

## Security Review Checklist

Before deploying to production:

- [ ] `TrustedProxies` contains only actual proxy IPs/CIDRs
- [ ] No public IP ranges in `TrustedProxies`
- [ ] `TrustProxyHeaders` is `false` if not behind proxy
- [ ] Tested that spoofed headers are ignored from untrusted sources
- [ ] Verified rate limiting triggers correctly
- [ ] Logging configured to detect suspicious patterns
- [ ] Documentation updated with current configuration

## Support

For security issues or questions, refer to:
- Security Fix Report: `SECURITY_FIX_REPORT.md`
- Main documentation: `/pkg/infra/middleware/ratelimit.go`
- Test examples: `/pkg/infra/middleware/ratelimit_test.go`
