"""Dados FICTÍCIOS para o servidor de referência do painel B.

Nada aqui vem do painel real: nomes, CNJ, valores e datas são inventados.
Os formatos seguem web/js/03-api-mapa.js e 04-api-carga.js do B.
"""
from datetime import date, timedelta


def _d(delta):
    return (date.today() + timedelta(days=delta)).isoformat()


from pessoas_processos import PESSOAS, PROCESSOS  # noqa: E402 (dados do Plano 2, iguais ao seed do v2)

from atendimentos_tarefas import ATENDIMENTOS, TAREFAS  # noqa: E402 (dados do Plano 3, iguais ao seed do v2)

AGENDA = [
    {"id": "ag-1", "tipo": "Reunião", "descricao": "Reunião com Maria Exemplo", "data": _d(2),
     "hora": "10:00", "local": "Escritório", "cliente": "Maria Exemplo"},
    {"id": "ag-2", "tipo": "Ligação", "descricao": "Retorno ao cliente João", "data": _d(1),
     "hora": "15:30", "local": "Telefone", "cliente": "João Fictício"},
    {"id": "ag-3", "tipo": "Aula", "descricao": "Aula de Direito Processual", "data": _d(3),
     "hora": "19:00", "local": "Faculdade Exemplo"},
    {"id": "ag-4", "tipo": "Prazo Fatal", "descricao": "Prazo fatal fictício", "data": _d(5),
     "hora": "", "local": "", "processoVinculado": "ativo/proc-0001.md"},
    {"id": "ag-5", "tipo": "Reunião", "descricao": "Reunião interna", "data": _d(6),
     "hora": "09:00", "local": "Sala 2"},
]

AUDIENCIAS = [
    {"arquivo": "instrucao-e-julgamento/aud-0001.md", "tipo": "instrucao-e-julgamento",
     "codigo": "AUD.0001", "title": "Instrução do caso Maria", "processo": "0000000-00.2026.8.26.0000",
     "cliente": "Maria Exemplo", "data": _d(9), "hora": "14:00", "juizo": "1ª Vara Cível de Exemplo",
     "local": "Fórum de Exemplo", "descricao": "Audiência fictícia."},
    {"arquivo": "conciliacao-e-mediacao/aud-0002.md", "tipo": "conciliacao-e-mediacao",
     "codigo": "AUD.0002", "title": "Conciliação Empresa Modelo", "processo": "0000001-01.2026.8.26.0001",
     "cliente": "Empresa Modelo Ltda", "data": _d(12), "hora": "10:30", "juizo": "2ª Vara de Exemplo",
     "local": "CEJUSC de Exemplo", "descricao": "Audiência fictícia."},
]

DILIGENCIAS = [
    {"arquivo": "dil-0001.md", "codigo": "DIL.0001", "title": "Buscar certidão no cartório",
     "processo": "0000000-00.2026.8.26.0000", "cliente": "Maria Exemplo", "local": "Cartório Exemplo",
     "data_limite": _d(4), "hora": "16:00", "responsavel": "Advogado Exemplo", "status": "Pendente",
     "descricao": "Diligência fictícia."},
    {"arquivo": "dil-0002.md", "codigo": "DIL.0002", "title": "Protocolo físico",
     "processo": "0000001-01.2026.8.26.0001", "cliente": "Empresa Modelo Ltda", "local": "Fórum de Exemplo",
     "data_limite": _d(8), "hora": "", "responsavel": "Estagiário Exemplo", "status": "Pendente",
     "descricao": "Diligência fictícia."},
]

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
