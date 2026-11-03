"""Pessoas e processos FICTÍCIOS do servidor de referência do painel B (Plano 2).

Os MESMOS dados existem no seed do v2 (internal/app/seed.go): a prova de paridade compara
as duas telas com conteúdo idêntico. Nada aqui vem do painel real.
Formato de saída: o que web/js/03-api-mapa.js (mapearPessoaAPI / mapearProcessoAPI) espera.
"""

PESSOAS = [
    {"arquivo": "maria-exemplo.md", "codigo": "PES. 0000001", "tipoPessoa": "Física",
     "nomeCompleto": "MARIA EXEMPLO", "cpf": "529.982.247-25", "rg": "12.345.678-9 SSP/SP",
     "estadoCivil": "Solteiro(a)", "profissao": "Professora", "nacionalidade": "Brasileiro(a)",
     "dataNascimento": "1985-04-12", "logradouro": "Rua das Flores", "numero": "100",
     "complemento": "Apto 12", "bairro": "Centro", "cep": "01001-000", "cidade": "São Paulo", "uf": "SP",
     "email": "maria@exemplo.test", "telefone": "(11) 91234-5678"},
    {"arquivo": "joao-ficticio.md", "codigo": "PES. 0000002", "tipoPessoa": "Física",
     "nomeCompleto": "JOÃO FICTÍCIO", "cpf": "111.444.777-35", "nacionalidade": "Brasileiro(a)",
     "cidade": "Rio de Janeiro", "uf": "RJ", "email": "joao@exemplo.test", "telefone": "(21) 98888-7777"},
    {"arquivo": "empresa-modelo-ltda.md", "codigo": "PES. 0000003", "tipoPessoa": "Jurídica",
     "razaoSocial": "EMPRESA MODELO LTDA", "nomeFantasia": "MODELO", "cnpj": "11.222.333/0001-81",
     "representanteNome": "Ana Representante", "representanteCpf": "123.456.789-09",
     "representanteCargo": "Sócia-administradora", "cidade": "Campinas", "uf": "SP",
     "email": "contato@modelo.test", "telefone": "(11) 3333-4444"},
    {"arquivo": "banco-de-teste-sa.md", "codigo": "PES. 0000004", "tipoPessoa": "Jurídica",
     "razaoSocial": "BANCO DE TESTE S.A.", "cnpj": "22.233.344/0001-83", "cidade": "Brasília", "uf": "DF"},
]

_PENDENCIAS = [
    ("numero_processo", "Número do processo (CNJ)"), ("sistema", "Sistema"), ("instancia", "Instância"),
    ("classe", "Classe processual"), ("ramo", "Matéria / ramo do direito"), ("comarca", "Comarca"),
    ("juizo", "Juízo"), ("distribuido_em", "Distribuição"), ("valor_causa", "Valor da causa"),
]


def _wiki(arquivo):
    return "[[pessoas/" + arquivo.replace(".md", "") + "]]"


def _processo(arquivo, codigo, estado, **campos):
    p = {"arquivo": arquivo, "codigo": codigo, "estado": estado, "title": "", "numero_processo": "",
         "sistema": "", "instancia": "", "juizo": "", "comarca": "", "classe": "", "ramo": "", "polo": "",
         "tipo_contingencia": "", "valor_causa": "", "distribuido_em": "", "citacao": "", "audiencia": "",
         "sentenca": "", "recursos": "", "fase_execucao": "", "observacoes": "", "partes": [],
         "responsaveis": [], "atendimento_origem": "", "projeto_vinculado": ""}
    p.update(campos)
    faltam = [rot for chave, rot in _PENDENCIAS if not p.get(chave)]
    if not any(pt.get("cliente") for pt in p["partes"]):
        faltam.append("Pelo menos uma parte marcada como cliente")
    p["camposFaltantes"] = faltam
    return p


PROCESSOS = [
    _processo("ativo/proc-0001.md", "PRO. 0000001", "ativo", title="MARIA EXEMPLO x BANCO DE TESTE S.A.",
              numero_processo="0000101-57.2025.8.26.0100", sistema="PJe", instancia="1º grau",
              juizo="1ª Vara Cível de Exemplo", comarca="São Paulo", classe="Procedimento Comum Cível",
              ramo="Consumidor", polo="Ativo", tipo_contingencia="Ativa", valor_causa="R$ 10.000,00",
              distribuido_em="2026-09-15", citacao="2026-09-30",
              partes=[{"pessoa": _wiki("maria-exemplo.md"), "papel": "Autor", "cliente": True},
                      {"pessoa": _wiki("banco-de-teste-sa.md"), "papel": "Réu", "cliente": False}],
              responsaveis=[_wiki("joao-ficticio.md")]),
    _processo("sobrestado/proc-0002.md", "PRO. 0000002", "sobrestado",
              title="EMPRESA MODELO LTDA x JOÃO FICTÍCIO", numero_processo="0000202-55.2024.5.02.0001",
              sistema="e-SAJ", instancia="1º grau", juizo="2ª Vara de Exemplo", comarca="Campinas",
              classe="Execução de Título Extrajudicial", ramo="Empresarial", valor_causa="R$ 50.000,00",
              distribuido_em="2026-10-20",
              partes=[{"pessoa": _wiki("empresa-modelo-ltda.md"), "papel": "Exequente", "cliente": True},
                      {"pessoa": _wiki("joao-ficticio.md"), "papel": "Executado", "cliente": False}]),
    _processo("extinto/proc-0003.md", "PRO. 0000003", "extinto", title="MARIA EXEMPLO x JOÃO FICTÍCIO",
              classe="Mandado de Segurança",
              partes=[{"pessoa": _wiki("maria-exemplo.md"), "papel": "Autor", "cliente": True},
                      {"pessoa": _wiki("joao-ficticio.md"), "papel": "Terceiro interessado", "cliente": False}]),
]
