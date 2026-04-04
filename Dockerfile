# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build backend (with embedded frontend)
FROM golang:1.23-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Copy frontend build output into the embed directory
COPY --from=frontend /app/frontend/dist ./internal/static/dist/
RUN CGO_ENABLED=1 go build -o /server ./cmd/server/

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend /server .
COPY config/config.example.yaml ./config/config.yaml
RUN mkdir -p data

EXPOSE 8080
CMD ["./server"]
