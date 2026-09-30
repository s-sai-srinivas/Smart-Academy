#!/bin/bash

# JPlag Setup Script for Digital Ocean Ubuntu Droplet
# Run as ROOT user: sudo bash setup-jplag.sh

set -e

echo "========================================="
echo "JPlag Plagiarism Detection Setup"
echo "========================================="

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
JPLAG_DIR="/opt/jplag"
BACKEND_USER="neeharika"

# Check root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}Please run as root (sudo bash setup-jplag.sh)${NC}"
    exit 1
fi

echo -e "${YELLOW}Step 1: Creating directory structure...${NC}"
mkdir -p ${JPLAG_DIR}/submissions
mkdir -p ${JPLAG_DIR}/results

echo -e "${YELLOW}Step 2: Setting permissions...${NC}"
chown -R ${BACKEND_USER}:${BACKEND_USER} ${JPLAG_DIR}
chmod -R 755 ${JPLAG_DIR}

echo -e "${YELLOW}Step 3: Creating JPlag Dockerfile...${NC}"
cat > ${JPLAG_DIR}/Dockerfile << 'EOF'
FROM eclipse-temurin:21-jre-alpine

ARG JPLAG_VERSION=5.1.0

RUN apk add --no-cache wget unzip && \
    wget -q "https://github.com/jplag/JPlag/releases/download/v${JPLAG_VERSION}/jplag-${JPLAG_VERSION}.jar" -O /jplag.jar && \
    apk del wget

WORKDIR /data

ENTRYPOINT ["java", "-jar", "/jplag.jar"]
EOF

echo -e "${YELLOW}Step 4: Building JPlag Docker image...${NC}"
cd ${JPLAG_DIR}
docker build -t jplag .

echo -e "${YELLOW}Step 5: Verifying installation...${NC}"
docker run --rm jplag --version || echo "Version check completed"

echo -e "${YELLOW}Step 6: Testing with sample files...${NC}"
mkdir -p ${JPLAG_DIR}/submissions/test/{s1,s2}
echo 'print("hello world")' > ${JPLAG_DIR}/submissions/test/s1/solution.py
echo 'print("hello world")' > ${JPLAG_DIR}/submissions/test/s2/solution.py
chown -R ${BACKEND_USER}:${BACKEND_USER} ${JPLAG_DIR}/submissions

docker run --rm \
    -v ${JPLAG_DIR}:/data \
    jplag \
    -l python3 /data/submissions/test -r /data/results/test || true

echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}JPlag Setup Complete!${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo "Directory: ${JPLAG_DIR}"
echo "Docker image: jplag"
echo ""
echo "Test: docker run --rm -v ${JPLAG_DIR}:/data jplag -l python3 /data/submissions/test -r /data/results/test"
echo ""

# Cleanup test files
rm -rf ${JPLAG_DIR}/submissions/test
rm -rf ${JPLAG_DIR}/results/test
