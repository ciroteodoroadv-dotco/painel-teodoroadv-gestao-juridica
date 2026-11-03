"""Atendimentos e tarefas FICTÍCIOS do servidor de referência do painel B (Plano 3).

Os MESMOS dados existem no seed do v2 (internal/app/seed.go): a prova de paridade compara as duas telas
com conteúdo idêntico. Nada aqui vem do painel real.
Formato de saída: o que web/js/03-api-mapa.js (mapearAtendimentoAPI / mapearTarefaAPI) espera.
A ordem das listas é a ordem de código, que é a que o v2 mostra.
"""

MARIA, JOAO = "[[pessoas/maria-exemplo]]", "[[pessoas/joao-ficticio]]"
EMPRESA, BANCO = "[[pessoas/empresa-modelo-ltda]]", "[[pessoas/banco-de-teste-sa]]"
P1, P2 = "ativo/proc-0001.md", "sobrestado/proc-0002.md"


def _atendimento(n, clientes, partes_adversas, descricao, data, criado_por="", observacoes=""):
    return {"arquivo": f"ate-{n:04d}.md", "codigo": f"ATE. {n:07d}", "title": "Atendimento",
            "atendimento_tipo": "Extrajudicial", "status": "Em triagem", "cliente": clientes[0], "clientes": clientes,
            "partes_adversas": partes_adversas, "criado_por": criado_por, "data_abertura": data,
            "descricao": descricao, "observacoes": observacoes}


ATENDIMENTOS = [
    _atendimento(1, [MARIA], [BANCO], "Revisão de contrato bancário (exemplo)", "2026-10-05", criado_por=JOAO),
    _atendimento(2, [EMPRESA], [], "Consulta sobre cobrança de duplicatas (exemplo)", "2026-10-20",
                 observacoes="Cliente pediu retorno por telefone."),
    _atendimento(3, [JOAO], [MARIA], "Triagem de caso trabalhista (exemplo)", "2026-11-02"),
]

_PARTES = [{"pessoa": MARIA, "papel": "Autor", "cliente": False}, {"pessoa": BANCO, "papel": "Réu", "cliente": False}]


def _tarefa(n, estado, titulo, origem_tipo, origem_id, prioridade, sub="", prazo="", fatal="", pai="", partes=None,
            responsaveis=None, criado_por="", descricao="", data_base=""):
    return {"arquivo": f"{estado}/tar-{n:04d}.md", "estado": estado, "codigo": f"TAR. {n:07d}", "title": titulo,
            "subtitulo": sub, "prioridade": prioridade, "prazo": prazo, "prazo_fatal": fatal, "data_base": data_base,
            "responsaveis": responsaveis or [], "criado_por": criado_por, "origem_tipo": origem_tipo,
            "origem_id": origem_id, "tarefa_pai": pai, "partes": partes or [], "observacoes": "", "descricao": descricao}


TAREFAS = [
    _tarefa(1, "pendente", "Protocolar embargos à execução (exemplo)", "processo", P1, "Urgente", prazo="2026-11-10",
            fatal="2026-11-12", partes=_PARTES, responsaveis=[JOAO], criado_por=JOAO, data_base="2026-11-03",
            descricao="Conferir os cálculos antes do protocolo."),
    _tarefa(2, "em-execucao", "Ligar para a cliente (exemplo)", "atendimento", "ate-0001.md", "Alta", sub="Ligação",
            prazo="2026-11-05", responsaveis=[JOAO]),
    _tarefa(3, "revisao", "Aula sobre execução (exemplo)", "atendimento", "ate-0002.md", "Normal", sub="Aula",
            prazo="2026-11-20"),
    _tarefa(4, "a-confirmar", "Conferir intimação (exemplo)", "processo", P2, "Normal", fatal="2026-11-15",
            responsaveis=[JOAO]),
    _tarefa(5, "concluidas-com-sucesso", "Enviar minuta ao cliente (exemplo)", "atendimento", "ate-0001.md", "Baixa"),
    _tarefa(6, "pendente", "Juntar procuração (exemplo)", "processo", P1, "Normal", fatal="2026-11-11",
            pai="TAR. 0000001", partes=_PARTES, responsaveis=[JOAO]),
]
