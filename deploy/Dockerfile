# Imagens fixadas por digest (índice multi-arquitetura: amd64 e arm64). Atualizar pelo Dependabot ou por `scripts/pinar-imagens.sh`.
FROM golang:1.26.10-bookworm@sha256:23ce71337b726608afb1bc00986e630a05034a87a87ea1fc2cd1ae5cf33387eb AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
COPY web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /painel ./cmd/painel && mkdir -p /data/documentos

# Imagem final sem shell, sem root.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
COPY --from=build /painel /painel
# pasta dos documentos, já pertencente ao usuário sem privilégios (o volume herda o dono)
COPY --from=build --chown=65532:65532 /data /data
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/painel"]
