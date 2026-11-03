"""Compromissos, audiências e diligências FICTÍCIOS do servidor de referência do painel B (Plano 4).

A MESMA massa existe no seed do v2 (internal/app/seed.go, semearAgenda). Nada aqui vem do painel real.
Formato: o que mapearAgendaAPI / mapearAudienciaAPI / mapearDiligenciaAPI (web/js/03-api-mapa.js) esperam.
"""

MARIA, JOAO = "[[pessoas/maria-exemplo]]", "[[pessoas/joao-ficticio]]"
EMPRESA = "[[pessoas/empresa-modelo-ltda]]"

AGENDA = [
    {"id": 1, "tipo": "Reunião", "descricao": "Reunião de alinhamento com a cliente (exemplo)", "data": "2026-11-10",
     "hora": "14:00", "local": "Escritório (exemplo)", "cliente": MARIA, "processoVinculado": "ativo/proc-0001.md",
     "criadoPor": JOAO},
    {"id": 2, "tipo": "Ligação", "descricao": "Ligação de retorno ao cliente (exemplo)", "data": "2026-11-11",
     "hora": "10:30", "local": "Telefone (exemplo)", "atendimentoVinculado": "ate-0002.md"},
    {"id": 3, "tipo": "Aula", "descricao": "Aula sobre processo civil (exemplo)", "data": "2026-11-12",
     "hora": "19:00", "local": "Sala virtual (exemplo)"},
]

AUDIENCIAS = [
    {"arquivo": "instrucao-e-julgamento/aud-0001.md", "tipo": "instrucao-e-julgamento", "codigo": "AUD. 0000001",
     "title": "Audiência de instrução — Maria Exemplo x Banco de Teste", "processo": "0000101-35.2025.8.26.0100",
     "cliente": MARIA, "data": "2026-11-13", "hora": "09:30", "juizo": "1ª Vara Cível de Exemplo",
     "local": "Fórum (exemplo), sala 3", "descricao": "Levar a procuração (exemplo)."},
    {"arquivo": "sustentacao-oral/aud-0002.md", "tipo": "sustentacao-oral", "codigo": "AUD. 0000002",
     "title": "Sustentação oral no recurso (exemplo)", "processo": "", "cliente": EMPRESA, "data": "2026-11-18",
     "hora": "14:00", "juizo": "Câmara de Exemplo", "local": "Tribunal (exemplo)", "descricao": "",
     "cancelada": "true", "motivo_cancelamento": "Remarcada pelo juízo (exemplo)"},
]

DILIGENCIAS = [
    {"arquivo": "dil-0001.md", "codigo": "DIL. 0000001", "title": "Protocolo presencial no fórum (exemplo)",
     "processo": "0000101-35.2025.8.26.0100", "cliente": MARIA, "local": "Fórum (exemplo)", "data_limite": "2026-11-14",
     "hora": "11:00", "responsavel": "Fulano Exemplo", "status": "Pendente", "descricao": ""},
    {"arquivo": "dil-0002.md", "codigo": "DIL. 0000002", "title": "Vistoria no imóvel (exemplo)",
     "processo": "", "cliente": "", "local": "Imóvel (exemplo)", "data_limite": "2026-11-20",
     "hora": "15:00", "responsavel": "Beltrano Fictício", "status": "Pendente", "descricao": ""},
]
