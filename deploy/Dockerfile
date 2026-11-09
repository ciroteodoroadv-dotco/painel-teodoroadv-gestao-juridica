# Imagens fixadas por digest (índice multi-arquitetura: amd64 e arm64). Atualizar pelo Dependabot ou por `scripts/pinar-imagens.sh`.
FROM golang:1.27.3-bookworm@sha256:dc0963f32053e4bc451d24d87e3892d42d01021b61f093e98eb95c0352bb84b0 AS build
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
