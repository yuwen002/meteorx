#!/bin/bash

# MeteorX Deployment Script
# Usage: ./deploy.sh [dev|prod]

set -e

ENV=${1:-prod}
COMPOSE_FILE="docker-compose.prod.yml"

echo "MeteorX Deployment Script"
echo "============================"
echo "Environment: $ENV"
echo ""

# Check if .env file exists
if [ ! -f .env ]; then
    echo "Warning: .env file not found!"
    echo "Creating from .env.example..."
    if [ -f .env.example ]; then
        cp .env.example .env
        echo "Created .env file from example."
        echo "Please edit .env file with your production values before continuing!"
        exit 1
    else
        echo ".env.example not found. Please create .env file manually."
        exit 1
    fi
fi

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check dependencies
echo "Checking dependencies..."
if ! command_exists docker; then
    echo "Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command_exists docker-compose; then
    echo "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

echo "All dependencies are installed."
echo ""

# Pull latest changes if git is available
if command_exists git && [ -d .git ]; then
    echo "Pulling latest changes from git..."
    git pull origin main || echo "Could not pull from git, continuing with local files..."
    echo ""
fi

# Build and start services
echo "Building and starting services..."
docker-compose -f $COMPOSE_FILE down --remove-orphans 2>/dev/null || true
docker-compose -f $COMPOSE_FILE pull
docker-compose -f $COMPOSE_FILE build --no-cache
docker-compose -f $COMPOSE_FILE up -d

echo ""
echo "Waiting for services to be healthy..."
sleep 10

# Check service status
echo ""
echo "Service Status:"
docker-compose -f $COMPOSE_FILE ps

echo ""
echo "Deployment completed successfully!"
echo ""
echo "Access your application:"
echo "   - Web Admin: http://localhost"
echo "   - API: http://localhost:8081"
echo ""
echo "Useful commands:"
echo "   - View logs: docker-compose -f $COMPOSE_FILE logs -f"
echo "   - Stop services: docker-compose -f $COMPOSE_FILE down"
echo "   - Restart: docker-compose -f $COMPOSE_FILE restart"
echo ""
