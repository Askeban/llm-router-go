# 🚀 LLM Router Website Deployment Guide

## 📋 Prerequisites

1. **Google Cloud Account** with billing enabled
2. **Google Cloud CLI** installed and configured
3. **Node.js 18+** and npm installed
4. **Domain** for custom URL (optional but recommended)

## 🛠️ Local Development

### 1. Install Dependencies
```bash
cd website
npm install
```

### 2. Run Development Server
```bash
npm run dev
```
Visit `http://localhost:3000` to see the website.

### 3. Build for Production
```bash
npm run build
```

## ☁️ Google Cloud Platform Deployment

### 1. Setup GCP Project
```bash
# Create new project (optional)
gcloud projects create llm-router-website --name="LLM Router Website"

# Set project
gcloud config set project llm-router-website

# Enable required APIs
gcloud services enable appengine.googleapis.com
gcloud services enable cloudbuild.googleapis.com
```

### 2. Initialize App Engine
```bash
gcloud app create --region=us-central1
```

### 3. Configure Environment Variables
Create `.env.production` file:
```bash
API_BASE_URL=https://api.llmrouter.ai
ANALYTICS_ID=G-XXXXXXXXXX
```

### 4. Deploy to App Engine
```bash
# Build the Next.js application
npm run build

# Deploy to App Engine
gcloud app deploy app.yaml --quiet
```

### 5. Set Custom Domain (Optional)
```bash
# Map custom domain
gcloud app domain-mappings create llmrouter.ai --certificate-management=AUTOMATIC

# Add DNS records (follow GCP instructions)
```

## 🔧 Configuration Options

### App Engine Settings
Edit `app.yaml`:
- **runtime**: Node.js version
- **scaling**: Auto-scaling parameters  
- **env_variables**: Environment configuration

### Next.js Configuration
Edit `next.config.js`:
- **API endpoints**: Backend service URLs
- **Build optimization**: Performance settings
- **Export settings**: Static generation options

## 📊 Monitoring & Analytics

### 1. Google Cloud Monitoring
```bash
# Enable monitoring
gcloud services enable monitoring.googleapis.com

# View logs
gcloud app logs tail -s default
```

### 2. Performance Monitoring
- **PageSpeed Insights**: Monitor loading performance
- **Google Analytics**: Track user engagement
- **Error Reporting**: Monitor application errors

## 🔐 Security Configuration

### 1. HTTPS Enforcement
App Engine automatically provides SSL certificates and redirects HTTP to HTTPS.

### 2. Security Headers
Add to `next.config.js`:
```javascript
module.exports = {
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
        ],
      },
    ]
  },
}
```

## 💰 Cost Optimization

### Expected Costs (Monthly)
- **App Engine**: $50-200 (based on traffic)
- **Cloud CDN**: $20-100 (bandwidth)
- **Domain**: $12/year
- **SSL Certificate**: Free (Google-managed)

### Optimization Tips
1. Use App Engine auto-scaling
2. Enable Cloud CDN for static assets
3. Implement proper caching headers
4. Monitor usage with Cloud Monitoring

## 🚀 CI/CD Pipeline (Optional)

### GitHub Actions Deployment
Create `.github/workflows/deploy.yml`:
```yaml
name: Deploy to Google App Engine

on:
  push:
    branches: [ main ]

jobs:
  deploy:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '18'
        
    - name: Install dependencies
      run: npm install
      working-directory: ./website
      
    - name: Build application
      run: npm run build
      working-directory: ./website
      
    - name: Deploy to App Engine
      uses: google-github-actions/deploy-appengine@v1
      with:
        deliverables: ./website/app.yaml
        project_id: ${{ secrets.GCP_PROJECT_ID }}
        credentials: ${{ secrets.GCP_SA_KEY }}
```

## 📱 Mobile Optimization

The website is built with mobile-first design:
- **Responsive layouts** adapt to all screen sizes
- **Touch-friendly interactions** for mobile users
- **Fast loading** optimized for mobile networks
- **Progressive enhancement** for feature detection

## 🔍 SEO Optimization

Built-in SEO features:
- **Meta tags** for social sharing
- **Structured data** for search engines
- **Sitemap generation** for indexing
- **Fast loading** for search ranking

## 🛟 Troubleshooting

### Common Issues

1. **Build Errors**
   ```bash
   # Clear Next.js cache
   rm -rf .next
   npm run build
   ```

2. **Deployment Failures**
   ```bash
   # Check App Engine logs
   gcloud app logs tail -s default
   ```

3. **Performance Issues**
   ```bash
   # Enable Cloud CDN
   gcloud compute backend-buckets create website-assets
   ```

### Support Resources
- **Google Cloud Documentation**: Cloud.google.com/appengine/docs
- **Next.js Documentation**: nextjs.org/docs
- **Community Support**: Stack Overflow, GitHub Issues

## ✅ Post-Deployment Checklist

- [ ] Website loads correctly at production URL
- [ ] All interactive features work (demo, calculator)
- [ ] Mobile responsiveness verified
- [ ] Performance metrics acceptable (<3s load time)
- [ ] Analytics tracking configured
- [ ] Error monitoring setup
- [ ] Domain and SSL configured
- [ ] Search engine indexing enabled

---

**Deployment Time**: ~15-30 minutes  
**Maintenance**: Minimal (auto-scaling, managed SSL)  
**Estimated Monthly Cost**: $70-300 (scales with traffic)