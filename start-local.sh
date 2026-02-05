#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Starting NFT Proxy Local Development Environment${NC}"
echo ""

# Create secrets directory if it doesn't exist
mkdir -p secrets

# Check and update .htpasswd with test:test credentials
echo -e "${YELLOW}🔑 Creating Basic Auth credentials (test:test)...${NC}"
if command -v htpasswd &> /dev/null; then
    htpasswd -cb secrets/.htpasswd test test
    echo -e "${GREEN}✅ Created .htpasswd with test:test credentials${NC}"
else
    echo -e "${RED}❌ htpasswd command not found. Please install apache2-utils (Debian/Ubuntu) or httpd-tools (RHEL/CentOS)${NC}"
    echo -e "${YELLOW}⚠️  Continuing with existing .htpasswd file (if any)...${NC}"
fi

# Stop and remove existing containers AND volumes
echo -e "${YELLOW}📦 Stopping existing containers...${NC}"
docker compose -f docker-compose-local.yml down --volumes

# Optional: remove unused volumes (for all projects)
echo -e "${YELLOW}🧹 Cleaning up unused volumes...${NC}"
docker volume prune -f

# Build and start containers
echo -e "${YELLOW}🔨 Building and starting containers...${NC}"
docker compose -f docker-compose-local.yml up --build -d

# Wait a moment for services to start
echo -e "${YELLOW}⏳ Waiting for services to start...${NC}"
sleep 5

# Check service health
echo ""
echo -e "${GREEN}✅ Local environment started successfully!${NC}"
echo ""
echo -e "${BLUE}📋 Available Services:${NC}"
echo -e "  🌐 NFT Proxy API:       http://localhost:8080"
echo -e "  🔐 Auth Service:        http://localhost:8081"
echo -e "  📊 Metrics:             http://localhost:8080/metrics"
echo -e "  📈 Grafana:             http://localhost:3000"
echo ""
echo -e "${GREEN}🔐 Authentication:${NC}"
echo -e "  📝 Basic Auth:          test:test"
echo -e "  🔑 Alchemy API Key:     dCv2CXvXaAMTLt5Meu_EHla3BNzLRTvt"
echo ""
echo -e "${GREEN}🧪 Quick Test Commands:${NC}"
echo -e "  • Run tests:            ${YELLOW}./test-local.sh${NC}"
echo -e "  • Health check:         ${YELLOW}curl http://localhost:8080/health${NC}"
echo -e "  • Test with Basic Auth: ${YELLOW}curl -u test:test 'http://localhost:8080/ethereum/mainnet/nft/v3/getOwnersForContract?contractAddress=0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D'${NC}"
echo -e "  • Get puzzle:           ${YELLOW}curl http://localhost:8080/auth/puzzle | jq${NC}"
echo -e "  • Check metrics:        ${YELLOW}curl http://localhost:8080/metrics${NC}"
echo -e "  • Check logs:           ${YELLOW}docker compose -f docker-compose-local.yml logs -f${NC}"
echo -e "  • Stop services:        ${YELLOW}docker compose -f docker-compose-local.yml down${NC}"
echo ""
echo -e "${BLUE}📖 Supported Chains:${NC}"
echo -e "  • ethereum/mainnet"
echo -e "  • polygon/mainnet"
echo -e "  • arbitrum/mainnet"
echo -e "  • optimism/mainnet"
echo -e "  • base/mainnet"
echo ""
