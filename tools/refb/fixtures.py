"""Dados FICTÍCIOS para o servidor de referência do painel B.

Nada aqui vem do painel real: nomes, CNJ, valores e datas são inventados.
Os formatos seguem web/js/03-api-mapa.js e 04-api-carga.js do B.
"""
from datetime import date, timedelta


def _d(delta):
    return (date.today() + timedelta(days=delta)).isoformat()


PESSOAS = [
    {"arquivo": "maria-exemplo.md", "codigo": "PES.0001", "tipoPessoa": "Pessoa Física",
     "nomeCompleto": "Maria Exemplo", "cpf": "529.982.247-25", "estadoCivil": "Solteiro(a)",
     "profissao": "Professora", "nacionalidade": "brasileira", "email": "maria@exemplo.test",
     "telefone": "(11) 91234-5678", "cidade": "São Paulo", "uf": "SP", "criado": _d(-30)},
    {"arquivo": "joao-ficticio.md", "codigo": "PES.0002", "tipoPessoa": "Pessoa Física",
     "nomeCompleto": "João Fictício", "cpf": "111.444.777-35", "email": "joao@exemplo.test",
     "telefone": "(21) 98888-7777", "cidade": "Rio de Janeiro", "uf": "RJ", "criado": _d(-25)},
    {"arquivo": "empresa-modelo-ltda.md", "codigo": "PES.0003", "tipoPessoa": "Pessoa Jurídica",
     "razaoSocial": "Empresa Modelo Ltda", "nomeFantasia": "Modelo", "cnpj": "11.222.333/0001-81",
     "representanteNome": "Ana Representante", "email": "contato@modelo.test",
     "telefone": "(11) 3333-4444", "cidade": "Campinas", "uf": "SP", "criado": _d(-20)},
    {"arquivo": "banco-de-teste-sa.md", "codigo": "PES.0004", "tipoPessoa": "Pessoa Jurídica",
     "razaoSocial": "Banco de Teste S.A.", "cnpj": "22.333.444/0001-55", "cidade": "Brasília",
     "uf": "DF", "criado": _d(-18)},
]

PROCESSOS = [
    {"arquivo": "ativo/proc-0001.md", "codigo": "PRO.0001", "title": "Maria Exemplo x Banco de Teste",
     "numero_processo": "0000000-00.2026.8.26.0000", "sistema": "PJe", "instancia": "1º Grau",
     "juizo": "1ª Vara Cível de Exemplo", "comarca": "São Paulo", "classe": "Procedimento Comum Cível",
     "ramo": "Consumidor", "polo": "Ativo", "valor_causa": "R$ 10.000,00", "distribuido_em": _d(-40),
     "estado": "ativo", "proximo_prazo": _d(6),
     "partes": [{"pessoa": "Maria Exemplo", "papel": "cliente", "cliente": True},
                {"pessoa": "Banco de Teste S.A.", "papel": "parte_contraria", "cliente": False}]},
    {"arquivo": "ativo/proc-0002.md", "codigo": "PRO.0002", "title": "Empresa Modelo – execução",
     "numero_processo": "0000001-01.2026.8.26.0001", "sistema": "e-SAJ", "instancia": "1º Grau",
     "juizo": "2ª Vara de Exemplo", "comarca": "Campinas", "classe": "Execução de Título Extrajudicial",
     "ramo": "Empresarial", "polo": "Ativo", "valor_causa": "R$ 50.000,00", "distribuido_em": _d(-15),
     "estado": "ativo",
     "partes": [{"pessoa": "Empresa Modelo Ltda", "papel": "cliente", "cliente": True},
                {"pessoa": "João Fictício", "papel": "parte_contraria", "cliente": False}]},
    {"arquivo": "sobrestado/proc-0003.md", "codigo": "PRO.0003", "title": "João Fictício – inventário",
     "numero_processo": "0000002-02.2025.8.19.0002", "sistema": "Projudi", "instancia": "1º Grau",
     "juizo": "Vara de Sucessões de Exemplo", "comarca": "Rio de Janeiro",
     "classe": "Outra (especificar)", "ramo": "Sucessões e Família", "polo": "Ativo",
     "valor_causa": "R$ 120.000,00", "distribuido_em": _d(-200), "estado": "sobrestado",
     "partes": [{"pessoa": "João Fictício", "papel": "cliente", "cliente": True}]},
]

ATENDIMENTOS = [
    {"arquivo": "ate-0001.md", "codigo": "ATE.0001", "title": "Consulta sobre contrato bancário",
     "atendimento_tipo": "Triagem", "cliente": "Maria Exemplo", "clientes": ["Maria Exemplo"],
     "status": "Em andamento", "data_abertura": _d(-45), "descricao": "Atendimento fictício."},
    {"arquivo": "ate-0002.md", "codigo": "ATE.0002", "title": "Revisão de cláusulas",
     "atendimento_tipo": "Extrajudicial", "cliente": "Empresa Modelo Ltda",
     "clientes": ["Empresa Modelo Ltda"], "status": "Em triagem", "data_abertura": _d(-10),
     "descricao": "Atendimento fictício."},
    {"arquivo": "ate-0003.md", "codigo": "ATE.0003", "title": "Orientação sucessória",
     "atendimento_tipo": "Triagem", "cliente": "João Fictício", "clientes": ["João Fictício"],
     "status": "Concluído", "data_abertura": _d(-90), "descricao": "Atendimento fictício."},
]


def _tarefa(arq, estado, titulo, sub, prio, prazo, fatal=""):
    return {"arquivo": f"{estado}/{arq}.md", "estado": estado, "codigo": "TAR." + arq[-4:],
            "title": titulo, "subtitulo": sub, "prioridade": prio, "prazo": prazo,
            "prazo_fatal": fatal, "data_base": _d(-5), "responsaveis": ["Advogado Exemplo"],
            "criado_por": "Advogado Exemplo", "origem_tipo": "processo",
            "origem_id": "ativo/proc-0001.md", "partes": [], "observacoes": "",
            "descricao": "Tarefa fictícia."}


TAREFAS = [
    _tarefa("tar-0001", "a-confirmar", "Conferir procuração", "Reunião", "Normal", _d(3)),
    _tarefa("tar-0002", "pendente", "Redigir contestação", "Aula", "Urgente", _d(4), _d(5)),
    _tarefa("tar-0003", "em-execucao", "Ligar para o cliente", "Ligação", "Alta", _d(1)),
    _tarefa("tar-0004", "revisao", "Revisar petição inicial", "", "Normal", _d(7)),
    _tarefa("tar-0005", "concluidas-com-sucesso", "Protocolar recurso", "", "Baixa", _d(-3)),
    _tarefa("tar-0006", "canceladas", "Tarefa cancelada de exemplo", "", "Baixa", _d(-8)),
]

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
