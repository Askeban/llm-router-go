# DNS Configuration Guide for RouteLLM.dev

## 🎯 Current Status

### ✅ **What's Working**
- **API Backend**: https://api.routellm.dev → `llm-router-api` (Cloud Run)
  - SSL: Active ✅
  - Health: Healthy ✅
  - Models: 26 active ✅

- **Backend Direct**: https://llm-router-api-717366250689.us-central1.run.app
  - Fully functional ✅

- **Frontend Direct**: https://llm-router-website-717366250689.us-central1.run.app
  - Serving static site ✅

### ⏳ **Waiting for DNS**
- **Frontend Domain**: routellm.dev → `llm-router-website`
  - Mapped in Cloud Run ✅
  - Needs DNS records configured ⏳

---

## 🔧 **DNS Configuration Required**

### **Option 1: Cloud Run Domain Mapping (RECOMMENDED ✅)**

This is **simpler** and provides **automatic SSL management**.

#### **Step 1: Update DNS Records at Your Registrar**

Go to your domain registrar (GoDaddy, Namecheap, Cloudflare, etc.) and add these records for **routellm.dev**:

```
Type    Name    Value
----    ----    -----
A       @       216.239.32.21
A       @       216.239.34.21
A       @       216.239.36.21
A       @       216.239.38.21
AAAA    @       2001:4860:4802:32::15
AAAA    @       2001:4860:4802:34::15
AAAA    @       2001:4860:4802:36::15
AAAA    @       2001:4860:4802:38::15
```

**Note**:
- `@` means root domain (routellm.dev)
- If your registrar doesn't support `@`, use `routellm.dev` or leave Name blank
- Keep existing `api.routellm.dev` records as they're already working

#### **Step 2: Verify DNS Propagation (15 mins - 48 hours)**

```bash
# Check DNS from command line
nslookup routellm.dev 8.8.8.8

# Should return one of: 216.239.32.21, 216.239.34.21, 216.239.36.21, 216.239.38.21
```

Or use online tools:
- https://dnschecker.org/#A/routellm.dev
- https://www.whatsmydns.net/#A/routellm.dev

#### **Step 3: SSL Certificate Auto-Provisioning**

Once DNS propagates:
- Google Cloud will automatically provision SSL certificate
- Usually takes **15-30 minutes** after DNS is live
- Check status:
  ```bash
  gcloud beta run domain-mappings describe --domain=routellm.dev --region=us-central1
  ```

---

## 🧹 **Old Infrastructure to Clean Up (Optional)**

You have an **old load balancer setup** that's no longer needed:

### **Resources Currently Unused:**
```
1. Load Balancer IP: 34.107.199.131 (routellm-https-ip)
2. SSL Certificate: routellm-ssl-final (FAILED - not in use)
3. URL Map: routellm-url-map
4. Backend Bucket: routellm-backend
5. HTTPS Proxy: routellm-https-proxy
6. HTTP Proxy: routellm-http-final
7. Forwarding Rules: routellm-https-forwarding-rule, routellm-http-final
```

### **Cost Impact:**
- **Static IP**: ~$3-5/month (reserved but not used)
- **Load Balancer**: ~$18/month (minimal usage)
- **Total waste**: ~$21-23/month

### **To Clean Up (saves ~$250/year):**

```bash
# 1. Delete forwarding rules
gcloud compute forwarding-rules delete routellm-https-forwarding-rule --global --quiet
gcloud compute forwarding-rules delete routellm-http-final --global --quiet

# 2. Delete proxies
gcloud compute target-https-proxies delete routellm-https-proxy --quiet
gcloud compute target-http-proxies delete routellm-http-final --quiet

# 3. Delete URL map
gcloud compute url-maps delete routellm-url-map --quiet

# 4. Delete backend bucket
gcloud compute backend-buckets delete routellm-backend --quiet

# 5. Delete SSL certificate (failed one)
gcloud compute ssl-certificates delete routellm-ssl-final --quiet

# 6. Release static IP
gcloud compute addresses delete routellm-https-ip --global --quiet
```

**⚠️ Warning**: Only run cleanup **AFTER** confirming Cloud Run domain mapping works!

---

## 📊 **Current Deployment Architecture**

```
┌─────────────────────────────────────────────────────────┐
│                    INTERNET                             │
└────────────────────┬───────────────┬────────────────────┘
                     │               │
                     │               │
            ┌────────▼────────┐     ┌▼──────────────────┐
            │ api.routellm.dev│     │  routellm.dev     │
            │    (Working ✅) │     │  (DNS Pending ⏳)  │
            └────────┬────────┘     └┬──────────────────┘
                     │               │
                     │               │
         ┌───────────▼──────────┐   ┌▼─────────────────────┐
         │  llm-router-api      │   │ llm-router-website   │
         │  (Cloud Run)         │   │ (Cloud Run)          │
         │  - 2Gi RAM           │   │ - 512Mi RAM          │
         │  - 1 CPU             │   │ - 1 CPU              │
         │  - 1-10 instances    │   │ - 0-5 instances      │
         │  - Auto SSL ✅       │   │ - Auto SSL (pending) │
         └───────────┬──────────┘   └──────────────────────┘
                     │
                     │
         ┌───────────▼──────────┐
         │  Cloud SQL Postgres  │
         │  llm-router-db       │
         │  (us-central1-c)     │
         └──────────────────────┘
```

---

## 🧪 **Testing After DNS Propagation**

### **1. Test Frontend**
```bash
curl -I https://routellm.dev/
# Should return: HTTP/2 200
```

### **2. Test API**
```bash
curl https://api.routellm.dev/health
# Should return: {"status":"healthy","models":26}
```

### **3. Test Registration (Beta Limit)**
```bash
curl -X POST https://api.routellm.dev/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "testpass123",
    "full_name": "Test User"
  }'

# First 100 users: Creates account
# After 100: Returns waitlist position
```

### **4. Test Waitlist**
```bash
curl -X POST https://api.routellm.dev/api/v1/auth/waitlist \
  -H "Content-Type: application/json" \
  -d '{
    "email": "waitlist@example.com",
    "full_name": "Waitlist User"
  }'

# Returns: {"message":"Added to waitlist successfully","position":1}
```

---

## 📋 **Action Checklist**

- [ ] **Update DNS records** at your domain registrar (see Option 1 above)
- [ ] **Wait 15 mins - 48 hours** for DNS propagation
- [ ] **Verify DNS** using dnschecker.org
- [ ] **Wait 15-30 mins** for SSL certificate provisioning
- [ ] **Test routellm.dev** - should load your website
- [ ] **Test api.routellm.dev** - should return healthy status
- [ ] **Clean up old load balancer** (optional, saves $250/year)

---

## 🆘 **Troubleshooting**

### **DNS Not Propagating?**
- **Check TTL**: Old DNS records may have high TTL (time to live)
- **Clear cache**: `sudo systemd-resolve --flush-caches` (Linux)
- **Use different DNS**: Test with `nslookup routellm.dev 8.8.8.8`

### **SSL Certificate Failing?**
```bash
# Check certificate status
gcloud beta run domain-mappings describe --domain=routellm.dev --region=us-central1

# Look for:
# status.conditions[?type=='CertificateProvisioned'].status: "True"
```

### **Website Not Loading?**
- Verify DNS points to Cloud Run IPs (216.239.x.x)
- Check SSL certificate is active
- Try direct Cloud Run URL first: https://llm-router-website-717366250689.us-central1.run.app

---

## ✅ **Summary**

**Current State:**
- ✅ API working on custom domain (api.routellm.dev)
- ✅ Frontend deployed and serving
- ⏳ Frontend custom domain waiting for DNS (routellm.dev)

**Next Steps:**
1. Configure DNS A/AAAA records (5 minutes work)
2. Wait for propagation (15 mins - 48 hours)
3. SSL auto-provisions (15-30 mins after DNS)
4. Clean up old infrastructure (optional)

**Total Cost After Cleanup:**
- Cloud Run API: ~$25-30/month
- Cloud Run Website: ~$5-10/month (scales to zero)
- Cloud SQL: ~$10/month (f1-micro)
- **Total: ~$40-50/month** (down from $60-75)

---

**🎉 Once DNS propagates, your entire system will be live on routellm.dev!**