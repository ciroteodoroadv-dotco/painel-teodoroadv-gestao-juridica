#!/usr/bin/env python3
"""Servidor de referência do painel B (somente leitura dos estáticos, dados fictícios).

Serve os arquivos de REFB_WEB e responde /api/* com fixtures inventadas.
Nunca executa o painel.exe, nunca abre dados/ nem .env, nunca grava nada.
"""
import argparse
import json
import mimetypes
import os
import sys
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import fixtures  # noqa: E402

PADRAO = ("/mnt/d/OneDrive/Desktop/painel-teodoro-entrega-1-1-2026-10-30/"
          "painel-teodoro/web")
WEB = Path(os.environ.get("REFB_WEB", PADRAO)).resolve()


class Manipulador(BaseHTTPRequestHandler):
    server_version = "refb/1"

    def log_message(self, *args):  # silencioso
        pass

    def _json(self, corpo, status=200):
        dados = json.dumps(corpo, ensure_ascii=False).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Cache-Control", "no-store")
        self.send_header("Content-Length", str(len(dados)))
        self.end_headers()
        self.wfile.write(dados)

    def _api(self, metodo):
        caminho = self.path.split("?", 1)[0]
        if metodo != "GET":
            n = int(self.headers.get("Content-Length") or 0)
            if n:
                self.rfile.read(n)  # descarta: nada é persistido
            return self._json({})
        if caminho == "/api/events":
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            try:
                while True:  # SSE vazio mantido aberto
                    self.wfile.write(b": keep-alive\n\n")
                    self.wfile.flush()
                    time.sleep(15)
            except (BrokenPipeError, ConnectionResetError):
                return
        chave = caminho[len("/api/"):].strip("/")
        if chave in fixtures.ROTAS:
            return self._json(fixtures.ROTAS[chave])
        return self._json({}, 404)

    def _estatico(self):
        caminho = self.path.split("?", 1)[0]
        if caminho == "/":
            caminho = "/index.html"
        alvo = (WEB / caminho.lstrip("/")).resolve()
        # Nunca sai da pasta web/ (impede dados/, .env etc.).
        if WEB not in alvo.parents or not alvo.is_file():
            self.send_response(404)
            self.end_headers()
            return
        dados = alvo.read_bytes()
        tipo = mimetypes.guess_type(str(alvo))[0] or "application/octet-stream"
        self.send_response(200)
        self.send_header("Content-Type", tipo + ("; charset=utf-8" if tipo.startswith("text/") or tipo.endswith("javascript") else ""))
        self.send_header("Cache-Control", "no-store")
        self.send_header("Content-Length", str(len(dados)))
        self.end_headers()
        self.wfile.write(dados)

    def do_GET(self):
        return self._api("GET") if self.path.startswith("/api/") else self._estatico()

    def do_POST(self):
        return self._api("POST")

    def do_PUT(self):
        return self._api("PUT")

    def do_PATCH(self):
        return self._api("PATCH")

    def do_DELETE(self):
        return self._api("DELETE")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--porta", type=int, default=18787)
    args = ap.parse_args()
    srv = ThreadingHTTPServer(("127.0.0.1", args.porta), Manipulador)
    print(f"refb em http://127.0.0.1:{args.porta}/  (web={WEB})", flush=True)
    srv.serve_forever()


if __name__ == "__main__":
    main()
