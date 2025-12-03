// API Configuration
//
// Embedded mode (Go binary): API_URL = "" (same origin)
// Vercel deployment: Edit API_URL below to point to your backend
//   Example: API_URL: "https://your-api.railway.app"
//
// Deploy to Vercel:
//   1. cd services/api-gateway/web
//   2. vercel deploy
//   3. Set API_URL to your backend URL
//
window.CONFIG = {
  // API base URL - empty for embedded, full URL for Vercel
  API_URL: "",

  // Endpoints
  get GRAPHQL_ENDPOINT() {
    return this.API_URL + "/graphql";
  },
  get PLAYGROUND_URL() {
    return this.API_URL + "/playground";
  },
  get REST_BASE() {
    return this.API_URL + "/api/v1";
  },
  get HEALTH_ENDPOINT() {
    return this.API_URL + "/api/v1/health";
  }
};
