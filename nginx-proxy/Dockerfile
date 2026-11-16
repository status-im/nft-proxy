# Build context: the repository root (proxy-common is a submodule under go-cache):
#   docker build -f nginx-proxy/Dockerfile .
FROM openresty/openresty:alpine

# Install dependencies
RUN apk add --no-cache ca-certificates curl perl && \
    update-ca-certificates && \
    /usr/local/openresty/bin/opm install ledgetech/lua-resty-http

# Copy configuration files
COPY nginx-proxy/nginx.conf /usr/local/openresty/nginx/conf/nginx.conf
COPY nginx-proxy/metrics.conf /usr/local/openresty/nginx/conf/metrics.conf
# Puzzle auth for nginx, shared with eth-rpc-proxy and market-proxy
COPY go-cache/proxy-common/auth/nginx/lua /usr/local/openresty/nginx/lua

# Expose port
EXPOSE 8080

# Start nginx
CMD ["openresty", "-g", "daemon off;"]
