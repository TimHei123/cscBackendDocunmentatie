#!/bin/bash
set -e

DOMAIN="timonheidenreich.nl"
EMAIL="your-email@example.com"  # Change to your email for certbot notifications

# Check for Docker
if ! command -v docker &>/dev/null; then
  echo "Docker not found. Installing Docker..."
  apt-get update
  apt-get install -y docker.io
fi

# Check for Docker Compose
if ! command -v docker-compose &>/dev/null; then
  echo "docker-compose not found. Installing docker-compose..."
  curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
  chmod +x /usr/local/bin/docker-compose
fi

# Create necessary directories if not exist
mkdir -p certbot/www/.well-known/acme-challenge
mkdir -p certbot/conf
mkdir -p swagger

# Check swagger.yaml presence
if [ ! -f "./swagger/swagger.yaml" ]; then
  echo "Please place your swagger.yaml inside ./swagger directory."
  exit 1
fi

echo "Starting Docker containers..."
docker-compose up -d

echo "Waiting for Nginx container to be ready..."
sleep 5

echo "Creating a test file for Let's Encrypt challenge..."
mkdir -p certbot/www/.well-known/acme-challenge
echo "letsencrypt-test" > certbot/www/.well-known/acme-challenge/testfile

echo "Verifying HTTP access to test file..."
curl --fail http://$DOMAIN/.well-known/acme-challenge/testfile

echo "Requesting Let's Encrypt certificate (this might take a moment)..."
docker-compose run --rm certbot certonly --webroot --webroot-path=/var/www/certbot -d $DOMAIN --email $EMAIL --agree-tos --no-eff-email

echo "Certificates obtained. Restarting nginx..."
docker-compose restart nginx

echo "Cleaning up test file..."
rm certbot/www/.well-known/acme-challenge/testfile

echo "Setup complete! Your site should be available at https://$DOMAIN/csc-self-servics/api-docs/"
