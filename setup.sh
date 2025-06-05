#!/bin/bash
set -e

# Define variables
DOMAIN="timonheidenreich.nl"
EMAIL="timon@timonheidenreich.eu"   # Replace with your email
WEBROOT="./certbot/www"          # Path mapped in docker-compose for /.well-known/acme-challenge

echo "Starting setup for domain: $DOMAIN"

# Step 1: Create necessary directories
echo "Creating directories for certbot webroot and config..."
mkdir -p "$WEBROOT"
mkdir -p ./certbot/conf

# Step 2: Create dummy test file for ACME challenge (optional, just to test nginx serving)
echo "Creating dummy ACME challenge test file..."
echo "testfile" > "$WEBROOT/testfile"

# Step 3: Start docker containers in detached mode
echo "Starting Docker Compose services..."
docker-compose up -d

# Step 4: Wait for nginx to be up and serving
echo "Waiting for nginx to be ready on port 80..."
until curl -s http://$DOMAIN/.well-known/acme-challenge/testfile | grep -q "testfile"; do
  echo "Waiting for nginx to serve the ACME challenge file..."
  sleep 3
done
echo "Nginx is serving challenge files correctly."

# Step 5: Remove the dummy test file (optional cleanup)
rm "$WEBROOT/testfile"

# Step 6: Run certbot to obtain or renew certificates
echo "Running certbot to obtain SSL certificates..."
docker-compose run --rm certbot certonly --webroot -w /var/www/certbot --email "$EMAIL" -d "$DOMAIN" --agree-tos --no-eff-email --force-renewal

# Step 7: Reload nginx to apply new certificates
echo "Reloading nginx to apply SSL certificates..."
docker-compose exec nginx nginx -s reload

echo "Setup complete. Your site should now have a valid SSL certificate."

