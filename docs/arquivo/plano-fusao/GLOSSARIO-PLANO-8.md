# Glossário — Plano 8

- **SGS**: Sistema Gerenciador de Séries Temporais do Banco Central; de lá vêm os índices da faixa do rodapé.
- **Série (SGS)**: um indicador com número próprio (ex.: 433 = IPCA). O painel lê o último ponto de cada série.
- **Data de referência**: o período a que o valor se refere (para índice mensal, o dia 1 do mês), não o dia em que o painel buscou.
- **Desatualizado**: valor cuja referência passou da validade (padrão 40 dias) ou cuja última busca falhou; continua visível, marcado.
- **ViaCEP**: serviço público e não oficial que devolve endereço a partir do CEP.
- **Cache (memória curta)**: guarda por 10 minutos a resposta de um CEP para não repetir a consulta.
- **Limite de taxa**: número máximo de consultas por minuto por usuário.
- **Pilha isolada**: cópia do painel (app + banco) em contêineres próprios, com banco descartável, para testar sem tocar no painel do advogado.
- **gosec / staticcheck / govulncheck**: ferramentas que leem o código procurando falha de segurança, erro de programação e dependência com vulnerabilidade conhecida.
- **Falso positivo**: aviso de ferramenta que, examinado, não é problema; fica registrado com a justificativa (`#nosec`).
- **Restauração**: usar o backup para recriar o banco e os documentos num lugar novo e conferir que voltaram iguais (contagens e hash).
- **Hash (sha256)**: impressão digital de um arquivo; se um byte mudar, o hash muda.
