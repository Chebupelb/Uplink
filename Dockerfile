FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./backend/cmd/server
RUN GOOS=js GOARCH=wasm go build -ldflags="-w -s" -o ./frontend/static/main.wasm ./frontend/main.go ./frontend/auth.go ./frontend/game.go ./frontend/lobby.go ./frontend/menu.go
RUN cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ./frontend/static/wasm_exec.js

FROM alpine:3.23
WORKDIR /app
RUN addgroup -S app && adduser -S user -G app
COPY --from=builder /app/server .
COPY backend/migrations ./migrations
COPY --from=builder /app/frontend/static ./frontend/static
USER user
EXPOSE 8080
CMD ["./server"]
