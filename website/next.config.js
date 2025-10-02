/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  trailingSlash: true,
  images: {
    unoptimized: true
  },
  env: {
    NEXT_PUBLIC_API_BASE_URL: process.env.NEXT_PUBLIC_API_BASE_URL || 'https://llm-router-api-717366250689.us-central1.run.app',
    ANALYTICS_ID: process.env.ANALYTICS_ID,
  }
}

module.exports = nextConfig