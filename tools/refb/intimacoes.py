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


# ---------------------------------------------------------------------------------------------------------------------
# Plano 6: monitor do DataJud. A MESMA massa existe em internal/app/seed_monitor.go (v2). Nada aqui vem do painel real.
# Processo cadastrado PRO. 0000001 (ativo); duas movimentações novas depois da linha de base, uma em cada grau.
# ---------------------------------------------------------------------------------------------------------------------
_NUM_P1 = cnj("0000101", "2025", "8", "26", "0100")
_PRAZO_INTERNO = "2026-11-05"  # criação (03/11/2026, São Paulo) + 2 dias corridos

# Mais recente primeiro (G2 foi registrada depois de G1). Em B a origem fica no campo "origem".
_DATAJUD = [
    ("dj-2", "Despacho", "G2"),
    ("dj-1", "Intimação", "G1"),
]
for _h, _tipo, _grau in _DATAJUD:
    INTIMACOES["intimacoes"].insert(len([x for x in INTIMACOES["intimacoes"] if x["origem"] == "DataJud"]), {
        "hash": _h, "origem": "DataJud", "tribunal": "TJSP", "orgao": "1ª Vara de Exemplo", "processo": _NUM_P1, "tipo": _tipo,
        "data_publ": "2026-11-02", "resumo": f"Grau: {_grau}\nMovimento: {_tipo}", "link": "", "status": "triagem"})

# Tarefa DJEN em triagem (exemplo-3) ganhou o prazo interno de confirmação (E3); as decididas o perderam.
for _t in TAREFAS_DJEN:
    if _t["origem_id"] == "exemplo-3":
        _t["prazo_fatal"] = _PRAZO_INTERNO

_DESC_JUD = ("Movimentação processual (DataJud). Esta fonte não fornece o teor da intimação.\n"
             "Prazo a apurar pelo advogado. O sistema não calculou prazo.\n"
             "Prazo interno de confirmação (2 dias corridos). NÃO é o prazo processual: o advogado apura o prazo da intimação.")


def _tarefa_jud(n, titulo, subtitulo, descricao):
    return {"arquivo": f"a-confirmar/tar-{n:04d}.md", "estado": "a-confirmar", "codigo": f"TAR. {n:07d}", "title": titulo,
            "subtitulo": subtitulo, "prioridade": "Normal", "prazo": "", "prazo_fatal": _PRAZO_INTERNO, "data_base": "",
            "responsaveis": [], "criado_por": "", "origem_tipo": "processo", "origem_id": "PRO. 0000001", "tarefa_pai": "",
            "partes": [], "observacoes": "", "descricao": descricao}


TAREFAS_DJEN += [
    _tarefa_jud(10, f"Analisar movimentação — Intimação — {_NUM_P1}", "Análise de decisão", _DESC_JUD),
    _tarefa_jud(11, f"Novidades do dia — {_NUM_P1}", "Análise consolidada do dia",
                "Novidades coletadas no dia.\nDataJud: Intimação (G1)\nDataJud: Despacho (G2)"),
    _tarefa_jud(12, f"Analisar movimentação — Despacho — {_NUM_P1}", "Análise de decisão", _DESC_JUD),
]

_MSG_MON = ("Processos: 3. Consultados com sucesso: 1. Sem registro ou parciais: 2. Sem número CNJ válido: 0. Falhas: 0. "
            "Movimentações novas: 2. Linhas de base registradas: 0.")
SINCRONIZACAO["fontes"] = [
    SINCRONIZACAO["fontes"][0],
    {"modulo": "monitor", "status": "parcial", "ultima_tentativa": _T, "ultimo_sucesso": None, "mensagem": _MSG_MON},
    {"modulo": "monitor-segredos", "status": "nao_sincronizado", "ultima_tentativa": _T, "ultimo_sucesso": None,
     "mensagem": "Nenhum processo em segredo de justiça cadastrado."},
]
