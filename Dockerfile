# 1. Build the React SPA
FROM node:24-alpine AS web
WORKDIR /web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 2. Build the Go binary with the SPA embedded
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o /plume ./cmd/plume

# 3. Minimal runtime image
FROM gcr.io/distroless/static-debian12
COPY --from=build /plume /plume
ENV PLUME_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/plume"]
