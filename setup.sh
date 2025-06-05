#!/bin/bash
set -e

# Check for local development mode
read -p "Is this a local development setup? (y/n): " is_local
if [ "$is_local" = "y" ]; then
    echo "Starting local development setup..."
    docker-compose up --build
    echo "Swagger UI is available at: http://localhost:8081"
    exit 0
fi

echo "Starting production setup..."

# Create necessary directories if they don't exist
mkdir -p certbot/conf certbot/www docs

# Read domain and email
read -p "Enter your domain name (e.g., api.example.com): " domain
read -p "Enter your email for SSL certificate: " email

# Replace domain in nginx.conf
sed -i "s/your-domain.com/$domain/g" nginx.conf

echo "Starting Nginx without SSL..."
docker-compose up --force-recreate -d nginx

echo "Requesting SSL certificate from Let's Encrypt..."
docker-compose run --rm certbot certonly --webroot -w /var/www/certbot -d "$domain" --email "$email" --agree-tos --no-eff-email

echo "Enabling SSL in nginx.conf..."
# Uncomment SSL server block (lines between "# SSL server block" comments)
sed -i '/# SSL server block/,/#}/ s/^#//' nginx.conf

echo "Restarting Nginx with SSL enabled..."
docker-compose restart nginx

echo "Setup complete! Your documentation should be available at:"
echo "https://$domain/"
echo "https://$domain/swagger/"
