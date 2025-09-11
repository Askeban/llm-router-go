/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  trailingSlash: true,
  images: {
    unoptimized: true
  },
  env: {
    API_BASE_URL: process.env.API_BASE_URL || 'http://localhost:8083',
    ANALYTICS_ID: process.env.ANALYTICS_ID,
  }
}

module.exports = nextConfig