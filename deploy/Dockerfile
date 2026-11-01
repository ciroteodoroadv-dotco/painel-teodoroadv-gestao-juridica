FROM golang:1.26.8-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
COPY web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /painel ./cmd/painel

# Imagem final sem shell, sem root.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /painel /painel
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/painel"]
