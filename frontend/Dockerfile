# ==============================================================================
# CogniShell Frontend UI - Dockerfile
# ==============================================================================

FROM node:18-alpine

WORKDIR /app

# Copy dependency manifests
COPY package*.json ./

# Install packages
RUN npm install

# Copy application source code
COPY . .

# Build Next.js application
RUN npm run build

# Expose Next.js default port
EXPOSE 3000

# Start UI server
CMD ["npm", "start"]
