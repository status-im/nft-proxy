FROM openresty/openresty:alpine

# Install dependencies
RUN apk add --no-cache ca-certificates curl perl && \
    update-ca-certificates && \
    /usr/local/openresty/bin/opm install ledgetech/lua-resty-http

# Copy configuration files
COPY nginx.conf /usr/local/openresty/nginx/conf/nginx.conf
COPY metrics.conf /usr/local/openresty/nginx/conf/metrics.conf
COPY lua /usr/local/openresty/nginx/lua

# Expose port
EXPOSE 8080

# Start nginx
CMD ["openresty", "-g", "daemon off;"]
