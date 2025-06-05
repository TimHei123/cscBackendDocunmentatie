#!/bin/bash

# Check if this is a local development setup
read -p "Is this a local development setup? (y/n): " is_local

if [ "$is_local" = "y" ]; then
    echo "Starting local development setup..."
    docker compose up --build
    echo "Swagger UI is available at: http://localhost:8081"
    exit 0
fi

# Production setup continues here
echo "Starting production setup..."

# Create necessary directories
mkdir -p certbot/conf
mkdir -p certbot/www
mkdir -p docs

# Copy documentation files
cp ../go/docs/API_Documentatie.pdf docs/
cp ../go/docs/static/api.html docs/
cp ../go/swagger.yaml ./

# Replace domain name in nginx.conf
read -p "Enter your domain name (e.g., api.example.com): " domain
sed -i "s/your-domain.com/$domain/g" nginx.conf

# Get email for SSL certificate
read -p "Enter your email for SSL certificate: " email

# Initial SSL certificate setup
docker-compose up --force-recreate -d nginx
docker-compose run --rm certbot certonly --webroot -w /var/www/certbot -d $domain --email $email --agree-tos --no-eff-email

# Restart nginx to apply SSL
docker-compose restart nginx

echo "Setup complete! Your documentation should be available at:"
echo "https://$domain/"
echo "https://$domain/swagger/" 