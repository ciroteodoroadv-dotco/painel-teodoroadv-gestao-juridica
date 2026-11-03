"""Dados FICTÍCIOS para o servidor de referência do painel B.

Nada aqui vem do painel real: nomes, CNJ, valores e datas são inventados.
Os formatos seguem web/js/03-api-mapa.js e 04-api-carga.js do B.
"""
from datetime import date, timedelta


def _d(delta):
    return (date.today() + timedelta(days=delta)).isoformat()


from pessoas_processos import PESSOAS, PROCESSOS  # noqa: E402 (dados do Plano 2, iguais ao seed do v2)

from atendimentos_tarefas import ATENDIMENTOS, TAREFAS  # noqa: E402 (dados do Plano 3, iguais ao seed do v2)

from agenda import AGENDA, AUDIENCIAS, DILIGENCIAS  # noqa: E402 (dados do Plano 4, iguais ao seed do v2)

INDICES = {
    "status": "sucesso", "origem": "BCB SGS", "data_ref": _d(-1),
    "indices": {
        "Selic (meta Copom)": {"rotulo": "Selic", "valor": "10,50% a.a.", "referencia": _d(-1)},
        "IPCA": {"rotulo": "IPCA", "valor": "0,40%", "referencia": _d(-1)},
        "INPC": {"rotulo": "INPC", "valor": "0,38%", "referencia": _d(-1)},
        "IGP-M": {"rotulo": "IGP-M", "valor": "0,25%", "referencia": _d(-1)},
        "Dólar (PTAX)": {"rotulo": "Dólar", "valor": "R$ 5,00", "referencia": _d(-1)},
        "Salário mínimo": {"rotulo": "Salário mínimo", "valor": "R$ 1.500,00", "referencia": _d(-1)},
    },
}

INTIMACOES = {
    "status": "sucesso",
    "intimacoes": [
        {"hash": "int-1", "origem": "DJEN", "tribunal": "TJXX", "orgao": "1ª Vara de Exemplo",
         "processo": "0000000-00.2026.8.26.0000", "tipo": "Intimação", "data_publ": _d(-1),
         "resumo": "Intimação fictícia.", "status": "triagem"},
        {"hash": "int-2", "origem": "DJEN", "tribunal": "TJXX", "orgao": "2ª Vara de Exemplo",
         "processo": "0000001-01.2026.8.26.0001", "tipo": "Despacho", "data_publ": _d(-2),
         "resumo": "Despacho fictício.", "status": "triagem"},
    ],
}

SINCRONIZACAO = {"fontes": [
    {"modulo": "intimacoes", "status": "nao_sincronizado", "mensagem": "Dado fictício de referência."},
]}

ROTAS = {
    "pessoas": PESSOAS, "processos": PROCESSOS, "tarefas": TAREFAS,
    "atendimentos": ATENDIMENTOS, "agenda": AGENDA, "audiencias": AUDIENCIAS,
    "diligencias": DILIGENCIAS, "indices": INDICES, "intimacoes": INTIMACOES,
    "sincronizacao/status": SINCRONIZACAO,
}
