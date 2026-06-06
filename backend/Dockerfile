FROM golang:1.26-alpine AS deps

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

FROM deps AS build

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bin/bootstrap-admin ./cmd/bootstrap-admin
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bin/migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bin/seed ./cmd/seed

FROM alpine:3.22 AS production

RUN apk add --no-cache ca-certificates tzdata \
	&& addgroup -S app \
	&& adduser -S -G app app

WORKDIR /app

COPY --from=build /out/bin /app/bin
COPY migrations /app/migrations

USER app

EXPOSE 8080

CMD ["/app/bin/api"]

FROM deps AS dev

COPY . .

EXPOSE 8080

CMD ["go", "run", "./cmd/api"]
