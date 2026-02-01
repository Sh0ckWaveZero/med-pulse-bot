#!/bin/bash

# Docker deployment script for Telegram Bot Med

set -e

echo "🚀 Telegram Bot Med - Docker Deployment"
echo "========================================"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️  .env file not found. Creating from .env.example...${NC}"
    cp .env.example .env
    echo -e "${RED}❌ Please edit .env file with your configuration before continuing.${NC}"
    exit 1
fi

# Load environment variables
export $(grep -v '^#' .env | xargs)

# Check required environment variables
if [ -z "$TELEGRAM_BOT_TOKEN" ] || [ "$TELEGRAM_BOT_TOKEN" = "your_telegram_bot_token_here" ]; then
    echo -e "${RED}❌ TELEGRAM_BOT_TOKEN is not set in .env file${NC}"
    exit 1
fi

if [ -z "$AUTHORIZED_CHAT_ID" ] || [ "$AUTHORIZED_CHAT_ID" = "your_chat_id_here" ]; then
    echo -e "${YELLOW}⚠️  AUTHORIZED_CHAT_ID is not set${NC}"
fi

echo -e "${GREEN}✅ Environment variables loaded${NC}"

# Function to display usage
usage() {
    echo "Usage: $0 {start|stop|restart|logs|status|update|clean}"
    echo ""
    echo "Commands:"
    echo "  start    - Start all services"
    echo "  stop     - Stop all services"
    echo "  restart  - Restart all services"
    echo "  logs     - View logs (add -f for follow mode)"
    echo "  status   - Check service status"
    echo "  update   - Update to latest version"
    echo "  clean    - Remove all containers and volumes"
    echo "  backup   - Backup database"
    echo "  restore  - Restore database from backup"
    exit 1
}

# Start services
start_services() {
    echo -e "${GREEN}🟢 Starting services...${NC}"
    docker-compose up -d
    echo -e "${GREEN}✅ Services started!${NC}"
    echo ""
    echo "App URL: http://localhost:8080"
    echo "Health Check: http://localhost:8080/health"
    echo ""
    echo "To view logs: $0 logs"
}

# Stop services
stop_services() {
    echo -e "${YELLOW}🛑 Stopping services...${NC}"
    docker-compose down
    echo -e "${GREEN}✅ Services stopped${NC}"
}

# Restart services
restart_services() {
    echo -e "${YELLOW}🔄 Restarting services...${NC}"
    docker-compose restart
    echo -e "${GREEN}✅ Services restarted${NC}"
}

# View logs
view_logs() {
    if [ "$2" = "-f" ] || [ "$2" = "--follow" ]; then
        docker-compose logs -f
    else
        docker-compose logs --tail=100
    fi
}

# Check status
check_status() {
    echo -e "${GREEN}📊 Service Status:${NC}"
    docker-compose ps
    echo ""
    echo -e "${GREEN}📈 Health Checks:${NC}"
    docker-compose exec -T app wget --no-verbose --tries=1 --spider http://localhost:8080/health 2>&1 | grep -q "OK" && echo -e "${GREEN}✅ App is healthy${NC}" || echo -e "${RED}❌ App health check failed${NC}"
}

# Update services
update_services() {
    echo -e "${YELLOW}⬆️  Updating services...${NC}"
    docker-compose pull
    docker-compose build --no-cache
    docker-compose up -d
    echo -e "${GREEN}✅ Services updated${NC}"
}

# Clean up
clean_services() {
    echo -e "${RED}⚠️  WARNING: This will remove all containers and volumes!${NC}"
    read -p "Are you sure? (yes/no): " confirm
    if [ "$confirm" = "yes" ]; then
        docker-compose down -v
        docker system prune -f
        echo -e "${GREEN}✅ Cleanup complete${NC}"
    else
        echo "Cancelled"
    fi
}

# Backup database (PocketBase)
backup_db() {
    BACKUP_DIR="./backups"
    mkdir -p $BACKUP_DIR
    BACKUP_FILE="$BACKUP_DIR/backup_$(date +%Y%m%d_%H%M%S).zip"
    
    echo -e "${GREEN}💾 Creating PocketBase backup...${NC}"
    # PocketBase data is in pb_data, just zip it
    # Note: In a real prod env, you might want to use the API or stop the service first
    zip -r $BACKUP_FILE pb_data
    echo -e "${GREEN}✅ Backup created: $BACKUP_FILE${NC}"
}

# Restore database
restore_db() {
    echo -e "${RED}❌ Restore not implemented for PocketBase script yet.${NC}"
    echo "Please manually unzip the backup to pb_data/"
}

# Main command handler
case "${1:-}" in
    start)
        start_services
        ;;
    stop)
        stop_services
        ;;
    restart)
        restart_services
        ;;
    logs)
        view_logs "$@"
        ;;
    status)
        check_status
        ;;
    update)
        update_services
        ;;
    clean)
        clean_services
        ;;
    backup)
        backup_db
        ;;
    restore)
        restore_db "$@"
        ;;
    *)
        usage
        ;;
esac
