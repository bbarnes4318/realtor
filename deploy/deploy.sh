#!/bin/bash
set -e

# Deployment configuration variables
DOMAIN="realtors.leadsbystorm.com"
IP_ADDR="87.99.155.241"
PROJECT_DIR="/var/www/realtor-scraper"

echo "=========================================================="
echo " Starting Realtor Scraper Remote Deployment Script"
echo " Target Domain: $DOMAIN"
echo " Target Server: $IP_ADDR"
echo "=========================================================="

# 1. Update packages and verify dependencies
echo "=> Updating system package lists..."
sudo apt-get update -y

install_if_missing() {
    PACKAGE=$1
    if ! dpkg -l | grep -q "^ii  $PACKAGE "; then
        echo "   Installing missing package: $PACKAGE..."
        sudo apt-get install -y $PACKAGE
    else
        echo "   Dependency satisfied: $PACKAGE"
    fi
}

install_if_missing "curl"
install_if_missing "git"
install_if_missing "nginx"
install_if_missing "certbot"
install_if_missing "python3-certbot-nginx"

# Install Docker if missing
if ! [ -x "$(command -v docker)" ]; then
    echo "=> Installing Docker CE..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    sudo usermod -aG docker $USER
    rm get-docker.sh
else
    echo "=> Docker CE is already installed."
fi

# Install Docker Compose if missing
if ! [ -x "$(command -v docker-compose)" ]; then
    echo "=> Installing Docker Compose..."
    sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
else
    echo "=> Docker Compose is already installed."
fi

# 2. Setup project folder structure
echo "=> Setting up project directory at $PROJECT_DIR..."
if [ ! -d "$PROJECT_DIR" ]; then
    sudo mkdir -p "$PROJECT_DIR"
    sudo chown -R $USER:$USER "$PROJECT_DIR"
    echo "=> Cloning repository..."
    git clone https://github.com/suffer-sami/realtor-scraper.git "$PROJECT_DIR"
else
    echo "=> Repository directory already exists. Fetching updates..."
    cd "$PROJECT_DIR"
    git pull origin main
fi

cd "$PROJECT_DIR"

# 3. Create .env file for Docker Compose
if [ ! -f ".env" ]; then
    echo "=> Setting up .env configuration..."
    cp .env.example .env
    
    # Prompt for JWT Secret or generate one
    read -p "Enter Realtor.com JWT_SECRET (or press Enter to generate a random secret): " USER_SECRET
    if [ -z "$USER_SECRET" ]; then
        USER_SECRET=$(openssl rand -hex 16)
        echo "Generated random JWT_SECRET: $USER_SECRET"
    fi
    
    # Modify .env file
    sed -i "s/JWT_SECRET=\"\"/JWT_SECRET=\"$USER_SECRET\"/g" .env
    sed -i "s/PLATFORM=\"dev\"/PLATFORM=\"prod\"/g" .env
    sed -i "s/USE_DB_LOCAL=true/USE_DB_LOCAL=true/g" .env
    sed -i "s/LOG_LEVEL=\"INFO\"/LOG_LEVEL=\"INFO\"/g" .env
else
    echo "=> Configuration .env file already exists."
fi

# Create frontend env file
if [ ! -f "frontend/.env" ]; then
    echo "NEXT_PUBLIC_API_URL=https://realtors.leadsbystorm.com" > frontend/.env
fi

# 4. Start Docker Compose Stack
echo "=> Building and starting Docker containers..."
sudo docker-compose down --remove-orphans || true
sudo docker-compose up --build -d

# 5. Configure Nginx Reverse Proxy
echo "=> Setting up Nginx virtual host..."
sudo cp deploy/nginx.conf /etc/nginx/sites-available/$DOMAIN
sudo ln -sf /etc/nginx/sites-available/$DOMAIN /etc/nginx/sites-enabled/

# Remove default site if it conflicts
sudo rm -f /etc/nginx/sites-enabled/default || true

# Test Nginx and Reload
echo "=> Verifying Nginx configurations..."
sudo nginx -t
sudo systemctl reload nginx

# 6. Apply Let's Encrypt SSL Cert via Certbot
echo "=> Requesting Let's Encrypt SSL certificate for $DOMAIN..."
echo "NOTE: Make sure your DNS A record points to $IP_ADDR before this step!"
sudo certbot --nginx -d $DOMAIN --non-interactive --agree-tos --email webmaster@leadsbystorm.com --redirect

echo "=========================================================="
echo " Deployment Complete!"
echo " Realtor Scraper is live on: https://realtors.leadsbystorm.com"
echo "=========================================================="
