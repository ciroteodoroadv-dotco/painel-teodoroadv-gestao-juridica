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

from intimacoes import INTIMACOES, SINCRONIZACAO, TAREFAS_DJEN  # noqa: E402 (dados do Plano 5, iguais ao seed do v2)

ROTAS = {
    "pessoas": PESSOAS, "processos": PROCESSOS, "tarefas": TAREFAS + TAREFAS_DJEN,
    "atendimentos": ATENDIMENTOS, "agenda": AGENDA, "audiencias": AUDIENCIAS,
    "diligencias": DILIGENCIAS, "indices": INDICES, "intimacoes": INTIMACOES,
    "sincronizacao/status": SINCRONIZACAO,
}
