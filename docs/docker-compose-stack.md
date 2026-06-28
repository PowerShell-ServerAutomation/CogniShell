version: '3.8'

services:

1. PostgreSQL (For relational state, script metadata, user mappings)

postgres:
image: postgres:15-alpine
container_name: cognishell-postgres
environment:
POSTGRES_USER: cognishell_admin
POSTGRES_PASSWORD: cognishell_password
POSTGRES_DB: cognishell_db
ports:
- "5432:5432"
volumes:
- postgres_data:/var/lib/postgresql/data
restart: unless-stopped
networks:
- cognishell-net

2. Grafana Loki (For 30-day ephemeral script execution logs)

loki:
image: grafana/loki:latest
container_name: cognishell-loki
ports:
- "3100:3100"
command: -config.file=/etc/loki/local-config.yaml
restart: unless-stopped
networks:
- cognishell-net

3. CogniShell Execution Engine (Go Backend)

cognishell-server:
build:
context: ./backend # Assuming backend code goes in a /backend directory
container_name: cognishell-server
ports:
- "8080:8080"
env_file:
- .env
depends_on:
- postgres
- loki
restart: on-failure
networks:
- cognishell-net

4. CogniShell UI (Next.js Frontend)

cognishell-ui:
build:
context: ./frontend # Assuming frontend code goes in a /frontend directory
container_name: cognishell-ui
ports:
- "3001:3000" # Mapped to 3001 on your host since Grafana is using 3000
environment:
- NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
depends_on:
- cognishell-server
restart: on-failure
networks:
- cognishell-net

volumes:
postgres_data:

networks:
cognishell-net:
driver: bridge