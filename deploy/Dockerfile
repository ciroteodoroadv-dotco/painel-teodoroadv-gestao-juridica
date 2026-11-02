FROM golang:1.26.10-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
COPY web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /painel ./cmd/painel && mkdir -p /data/documentos

# Imagem final sem shell, sem root.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /painel /painel
# pasta dos documentos, já pertencente ao usuário sem privilégios (o volume herda o dono)
COPY --from=build --chown=65532:65532 /data /data
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/painel"]
