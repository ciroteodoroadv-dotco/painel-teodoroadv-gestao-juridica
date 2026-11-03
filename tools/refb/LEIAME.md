# Servidor de referência do painel B

Serve os estáticos do painel B (somente leitura) e responde `/api/*` com dados
**fictícios** (`fixtures.py`). Não executa `painel.exe`, não abre `dados/` nem `.env`,
não grava nada (POST/PUT/PATCH/DELETE devolvem `{}`).

    python3 tools/refb/servidor.py --porta 18787
    REFB_WEB=/outro/caminho/web python3 tools/refb/servidor.py

Serve só para provar a paridade visual (`docs/plano-fusao/paridade/`). Fica fora do binário.
