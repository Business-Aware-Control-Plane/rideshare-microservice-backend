const getBackendUrls = () => {
  if (typeof window !== 'undefined') {
    const hostname = window.location.hostname;
    const port = window.location.port;

    // If accessing via local DNS ingress domains (*.rideshare)
    if (hostname.endsWith('.rideshare') || hostname === 'web.rideshare') {
      const protocol = window.location.protocol;
      const wsProtocol = protocol === 'https:' ? 'wss:' : 'ws:';
      const gatewayHost = 'api-gateway.rideshare';
      return {
        apiUrl: `${protocol}//${gatewayHost}${port ? `:${port}` : ''}`,
        wsUrl: `${wsProtocol}//${gatewayHost}${port ? `:${port}` : ''}/ws`
      };
    }
  }

  // Fallbacks for Server-Side Rendering (SSR) or direct local development port-forwarding
  return {
    apiUrl: process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8081',
    wsUrl: process.env.NEXT_PUBLIC_WEBSOCKET_URL ?? 'ws://localhost:8081/ws'
  };
};

const { apiUrl, wsUrl } = getBackendUrls();

export const API_URL = apiUrl;
export const WEBSOCKET_URL = wsUrl;
