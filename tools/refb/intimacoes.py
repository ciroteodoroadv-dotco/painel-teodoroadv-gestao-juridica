"""Intimações, tarefas de análise e estado das coletas FICTÍCIOS do servidor de referência do painel B (Plano 5).

A MESMA massa existe no seed do v2 (internal/app/seed_intimacoes.go). Nada aqui vem do painel real.
Formato: o que mapearIntimacaoAPI (web/js/03-api-mapa.js) e 16-sync-status.js do B esperam.
A ordem é a que o v2 mostra: data de publicação decrescente e, no mesmo dia, a mais recente primeiro.
"""


def cnj(n7, aaaa, j, tr, oooo):
    """Número único com dígito verificador (mesma conta de br.MakeCNJ no v2)."""
    dv = 98 - int(n7 + aaaa + j + tr + oooo + "00") % 97
    return f"{n7}-{dv:02d}.{aaaa}.{j}.{tr}.{oooo}"


# (n, tipo, número do processo, data da publicação, status)
_BASE = [
    (3, "Intimação", cnj("0000303", "2026", "8", "26", "0100"), "2026-11-02", "triagem"),
    (2, "Despacho", cnj("0000302", "2026", "8", "26", "0100"), "2026-11-02", "confirmada"),
    (1, "Sentença", cnj("0000301", "2026", "8", "26", "0100"), "2026-10-30", "descartada"),
]

DESCRICAO = "Prazo a apurar pelo advogado. O sistema não calculou prazo.\nLink: https://exemplo.test/djen/exemplo-%d"

INTIMACOES = {
    "status": "sucesso",
    "intimacoes": [
        {"hash": f"exemplo-{n}", "origem": "DJEN", "tribunal": "TJXX", "orgao": "1ª Vara de Exemplo", "processo": num,
         "tipo": tipo, "data_publ": data, "resumo": f"{tipo} fictícia (exemplo).",
         "link": f"https://exemplo.test/djen/exemplo-{n}", "status": status}
        for n, tipo, num, data, status in _BASE
    ],
}

# Tarefas de análise (TAR. 0000007 a 0000009, na ordem de criação: exemplo-1, -2, -3). Em B a origem é "DJEN" + o hash.
TAREFAS_DJEN = [
    {"arquivo": f"a-confirmar/tar-{6 + n:04d}.md", "estado": "a-confirmar", "codigo": f"TAR. {6 + n:07d}",
     "title": f"Analisar intimação — {tipo} — {num}", "subtitulo": "Análise de decisão", "prioridade": "Normal",
     "prazo": "", "prazo_fatal": "", "data_base": "", "responsaveis": [], "criado_por": "", "origem_tipo": "DJEN",
     "origem_id": f"exemplo-{n}", "tarefa_pai": "", "partes": [], "observacoes": "", "descricao": DESCRICAO % n}
    for n, tipo, num, _data, _status in sorted(_BASE)
]

_MSG = "OABs: 1. Comunicações novas: 3. Já registradas: 0. OABs com falha: 0. OABs recusadas: 0. Itens não registrados: 0."
_T = "2026-11-03T15:00:00Z"
SINCRONIZACAO = {"fontes": [
    {"modulo": "intimacoes", "status": "sucesso", "ultima_tentativa": _T, "ultimo_sucesso": _T, "mensagem": _MSG},
    {"modulo": "monitor", "status": "falha", "ultima_tentativa": _T, "ultimo_sucesso": None, "mensagem": "Dado fictício de exemplo."},
]}
