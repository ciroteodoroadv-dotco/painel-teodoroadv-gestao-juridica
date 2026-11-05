// Camada compartilhada por todos os módulos do painel.
// Qualquer módulo (Pessoas, Processos, Tarefas, Atendimentos) usa este
// mesmo arquivo para ler/gravar dados e para saber, em tempo real,
// quando outro módulo mudou algo.

const Painel = (() => {
  const listeners = {};

  function ao(entidade, callback) {
    if (!listeners[entidade]) listeners[entidade] = [];
    listeners[entidade].push(callback);
  }

  async function carregar(entidade) {
    const resp = await fetch(`/api/${entidade}`);
    if (!resp.ok) throw new Error(`Falha ao carregar ${entidade}`);
    return resp.json();
  }

  // Lê envelopes de sincronização mesmo quando a API devolve um status HTTP
  // de falha, permitindo que a interface mostre o estado operacional real.
  async function carregarResultado(url) {
    const resp = await fetch(url);
    const corpo = await resp.json().catch(() => ({}));
    if (!resp.ok) {
      const erro = new Error(corpo.erro || corpo.mensagem || `Falha ao carregar ${url}`);
      erro.dados = corpo;
      throw erro;
    }
    return corpo;
  }

  async function atualizarStatus(entidade, id, status) {
    const resp = await fetch(`/api/${entidade}/${encodeURIComponent(id)}/status`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status }),
    });
    const corpo = await resp.json().catch(() => ({}));
    if (!resp.ok) throw new Error(corpo.erro || `Falha ao atualizar ${entidade}`);
    return corpo;
  }

  async function salvar(entidade, lista) {
    const resp = await fetch(`/api/${entidade}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(lista),
    });
    if (!resp.ok) throw new Error(`Falha ao salvar ${entidade}`);
  }

  // Ação nomeada sobre um registro específico — usada quando a gravação tem
  // efeito colateral estrutural (ex.: mover uma ficha de processo entre
  // pastas de estado), e não pode ser tratada como "regravar a lista
  // inteira". estadoEArquivo é o par ["01-ativos", "nome-do-arquivo.md"].
  async function acao(entidade, [estado, arquivo], nomeAcao, payload, metodo = "PATCH", headers = {}) {
    const resp = await fetch(`/api/${entidade}/${estado}/${arquivo}/${nomeAcao}`, {
      method: metodo,
      headers: { "Content-Type": "application/json", ...headers },
      body: JSON.stringify(payload),
    });
    if (!resp.ok) {
      const texto = await resp.text().catch(() => "");
      throw new Error(`Falha em ${entidade}/${nomeAcao}: ${texto || resp.status}`);
    }
    return resp.json().catch(() => ({}));
  }

  function conectarTempoReal() {
    const es = new EventSource("/api/events");
    es.onmessage = (evento) => {
      const entidade = evento.data;
      (listeners[entidade] || []).forEach((cb) => cb());
    };
    // EventSource já reconecta automaticamente se a conexão cair.
  }

  conectarTempoReal();

  return { ao, carregar, carregarResultado, salvar, acao, atualizarStatus };
})();
