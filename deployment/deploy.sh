#!/bin/bash

# Samrt Academy Deployment Script for Digital Ocean Droplet
# This script sets up the complete samrt academy environment

set -e

echo "========================================="
echo "Samrt Academy Deployment Script"
echo "========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
DB_NAME="coding_platform"
DB_USER="coding_user"
DB_PASSWORD="your_secure_password_here"  # CHANGE THIS!
JWT_SECRET="your_jwt_secret_key_here"    # CHANGE THIS!
JUDGE0_URL="http://localhost:2358"

echo -e "${YELLOW}Step 1: Updating system packages...${NC}"
sudo apt update && sudo apt upgrade -y

echo -e "${YELLOW}Step 2: Installing PostgreSQL...${NC}"
sudo apt install postgresql postgresql-contrib -y
sudo systemctl start postgresql
sudo systemctl enable postgresql

echo -e "${YELLOW}Step 3: Setting up database...${NC}"
sudo -u postgres psql << EOF
CREATE DATABASE ${DB_NAME};
CREATE USER ${DB_USER} WITH ENCRYPTED PASSWORD '${DB_PASSWORD}';
GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};
\c ${DB_NAME}
GRANT ALL ON SCHEMA public TO ${DB_USER};
EOF

echo -e "${GREEN}Database created successfully!${NC}"

echo -e "${YELLOW}Step 4: Installing Nginx...${NC}"
sudo apt install nginx -y
sudo systemctl start nginx
sudo systemctl enable nginx

echo -e "${YELLOW}Step 5: Installing Go...${NC}"
if ! command -v go &> /dev/null; then
    wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin
    rm go1.21.6.linux-amd64.tar.gz
    echo -e "${GREEN}Go installed successfully!${NC}"
else
    echo -e "${GREEN}Go is already installed!${NC}"
fi

echo -e "${YELLOW}Step 6: Creating project directory...${NC}"
sudo mkdir -p /opt/coding-platform
sudo chown -R $USER:$USER /opt/coding-platform

echo -e "${YELLOW}Step 7: Copying project files...${NC}"
# Assuming you're running this from the project root
cp -r backend /opt/coding-platform/
cp -r frontend /opt/coding-platform/

echo -e "${YELLOW}Step 8: Creating .env file...${NC}"
cat > /opt/coding-platform/backend/.env << EOF
PORT=8080
GIN_MODE=release

DB_HOST=localhost
DB_PORT=5432
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}

JWT_SECRET=${JWT_SECRET}

JUDGE0_URL=${JUDGE0_URL}
EOF

echo -e "${GREEN}.env file created!${NC}"

echo -e "${YELLOW}Step 9: Installing Go dependencies...${NC}"
cd /opt/coding-platform/backend
go mod download
go mod tidy

echo -e "${YELLOW}Step 10: Configuring Nginx...${NC}"
sudo cp /opt/coding-platform/deployment/nginx.conf /etc/nginx/sites-available/coding-platform
sudo ln -sf /etc/nginx/sites-available/coding-platform /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx

echo -e "${YELLOW}Step 11: Setting up systemd service...${NC}"
sudo cp /opt/coding-platform/deployment/coding-platform.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable coding-platform
sudo systemctl start coding-platform

echo -e "${YELLOW}Step 12: Configuring firewall...${NC}"
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw --force enable

echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}Deployment completed successfully!${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo -e "Backend service status:"
sudo systemctl status coding-platform --no-pager
echo ""
echo -e "${YELLOW}Important:${NC}"
echo -e "1. Update the API_BASE_URL in frontend/js/api.js to your droplet IP or domain"
echo -e "2. Database password: ${DB_PASSWORD}"
echo -e "3. JWT secret: ${JWT_SECRET}"
echo -e "4. Access your app at: http://YOUR_DROPLET_IP"
echo ""
echo -e "${YELLOW}Useful commands:${NC}"
echo -e "  View backend logs: sudo journalctl -u coding-platform -f"
echo -e "  Restart backend: sudo systemctl restart coding-platform"
echo -e "  Reload Nginx: sudo systemctl reload nginx"
