package app

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/intimacoes"
	"painelteodoro/internal/monitor"
	"painelteodoro/internal/platform/webx"
)

const (
	numA = "0000101-57.2025.8.26.0100" // TJSP, DV calculado à mão (bc)
	numB = "1234567-37.2024.5.02.0001" // TRT2
)

// falsoDataJud é o DataJud dos testes: um servidor httptest que devolve o JSON configurado por número de processo.
type falsoDataJud struct {
	mu      sync.Mutex
	srv     *httptest.Server
	hits    map[string]string // dígitos do número -> JSON de "hits"
	chamou  int
	paths   []string
	trava   chan struct{}
	entrou  chan struct{}
	statusH int
}

func novoFalsoDataJud(t *testing.T) *falsoDataJud {
	f := &falsoDataJud{hits: map[string]string{}}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 2048)
		n, _ := r.Body.Read(b)
		corpo := string(b[:n])
		f.mu.Lock()
		f.chamou++
		f.paths = append(f.paths, r.URL.Path)
		trava, entrou, st := f.trava, f.entrou, f.statusH
		var resp string
		for dig, h := range f.hits {
			if strings.Contains(corpo, dig) {
				resp = h
			}
		}
		f.mu.Unlock()
		if entrou != nil {
			select {
			case entrou <- struct{}{}:
			default:
			}
		}
		if trava != nil {
			<-trava
		}
		if st != 0 {
			w.WriteHeader(st)
			return
		}
		if resp == "" {
			resp = `{"hits":{"hits":[]}}`
		}
		_, _ = w.Write([]byte(resp))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *falsoDataJud) n() int { f.mu.Lock(); defer f.mu.Unlock(); return f.chamou }

func (f *falsoDataJud) poe(numero string, hit string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hits[strings.NewReplacer("-", "", ".", "").Replace(numero)] = `{"hits":{"hits":[` + hit + `]}}`
}

func hitG(grau string, movs ...string) string {
	return `{"_source":{"tribunal":"TJSP","grau":"` + grau + `","orgaoJulgador":{"nome":"1ª Vara"},"movimentos":[` + strings.Join(movs, ",") + `]}}`
}

func mov(codigo int, nome, dataHora string) string {
	return fmt.Sprintf(`{"codigo":%d,"nome":%q,"dataHora":%q,"complementosTabelados":[{"codigo":1,"nome":"prazo","valor":"5","descricao":"dias"}]}`, codigo, nome, dataHora)
}

type ambMon struct {
	*env
	dj  *falsoDataJud
	ac  webx.Actor
	adv *browser
	svc *monitor.Service
}

func ambMonitor(t *testing.T, dj *falsoDataJud, ativo bool, chave string, opts ...Option) *ambMon {
	t.Helper()
	agora := time.Date(2026, 11, 4, 15, 0, 0, 0, time.UTC)
	opts = append([]Option{WithRelogio(func() time.Time { return agora }),
		WithDataJud(DataJud{Ativo: ativo, BaseURL: dj.srv.URL, PermitirHTTP: true, Chave: func() string { return chave }, AlertaHoras: 36})}, opts...)
	e := newEnvOpts(t, opts...)
	_, uid := e.user("adm-mon@exemplo.test", auth.Administrador)
	adv, _ := e.user("adv-mon@exemplo.test", auth.Advogado)
	return &ambMon{env: e, dj: dj, ac: webx.Actor{UserID: uid, RequestID: "t"}, adv: adv, svc: e.svc.Monitor}
}

// novosItens conta o que a coleta gera para o advogado (linhas de Intimações e tarefas de análise).
func (a *ambMon) automaticas() (intim, indiv, consol int) {
	return a.count(`SELECT count(*) FROM intimacoes WHERE origem='DataJud'`),
		a.count(`SELECT count(*) FROM tarefas WHERE intimacao_id IS NOT NULL`),
		a.count(`SELECT count(*) FROM tarefas WHERE consolidada_dia IS NOT NULL`)
}

func TestMonitorPrimeiraLeituraSoGravaBaseDepoisSoNovidades(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, true, "k")
	pid := a.adv.criarProcesso(t, numA)
	ctx := context.Background()
	dj.poe(numA, hitG("G1", mov(26, "Distribuição", "2025-03-10T10:00:00.000Z")))
	// (a) primeira leitura: só base.
	if _, err := a.svc.ColetarAgora(ctx, a.ac); err != nil {
		t.Fatal(err)
	}
	if i, ti, c := a.automaticas(); i != 0 || ti != 0 || c != 0 {
		t.Fatalf("(a) primeira leitura não gera linha nem tarefa: %d %d %d", i, ti, c)
	}
	if a.count(`SELECT count(*) FROM processo_movimentos WHERE processo_id=$1::uuid AND base AND fonte='DataJud'`, pid) != 1 ||
		a.count(`SELECT count(*) FROM monitor_processos WHERE processo_id=$1::uuid AND base_registrada_em IS NOT NULL AND status='sucesso'`, pid) != 1 {
		t.Error("(a) a linha de base deveria estar registrada")
	}
	// (b) segunda leitura: 2 movimentos novos.
	dj.poe(numA, hitG("G1", mov(26, "Distribuição", "2025-03-10T10:00:00.000Z"), mov(123, "Intimação", "2026-11-03T09:30:00.000Z"), mov(11, "Juntada", "2026-11-03T10:00:00.000Z")))
	if _, err := a.svc.ColetarAgora(ctx, a.ac); err != nil {
		t.Fatal(err)
	}
	if i, ti, c := a.automaticas(); i != 2 || ti != 2 || c != 1 {
		t.Fatalf("(b) 2 linhas, 2 tarefas individuais e 1 consolidada: %d %d %d", i, ti, c)
	}
	if a.count(`SELECT count(*) FROM tarefas WHERE (intimacao_id IS NOT NULL OR consolidada_dia IS NOT NULL) AND estado='a-confirmar' AND prazo_interno AND prazo_fatal = DATE '2026-11-06'`) != 3 {
		t.Error("(b)/(m) as 3 tarefas nascem em A confirmar com prazo interno de 2 dias (E3)")
	}
	if a.count(`SELECT count(*) FROM prazos WHERE tarefa_id IS NOT NULL AND situacao='a_conferir' AND data_fatal = DATE '2026-11-06'`) != 3 {
		t.Error("(m) o prazo interno fica a conferir no domínio Prazos")
	}
	var texto string
	_ = a.pool.QueryRow(ctx, `SELECT texto FROM intimacoes WHERE tipo='Intimação'`).Scan(&texto)
	for _, quer := range []string{"Grau: G1", "Órgão julgador: 1ª Vara", "Código do movimento: 123", "Movimento: Intimação", "03/11/2026 06:30", "prazo", "dias"} {
		if !strings.Contains(texto, quer) {
			t.Errorf("(E2) o texto do movimento deveria trazer %q:\n%s", quer, texto)
		}
	}
	// (c) terceira leitura igual: nada novo.
	if _, err := a.svc.ColetarAgora(ctx, a.ac); err != nil {
		t.Fatal(err)
	}
	if i, ti, c := a.automaticas(); i != 2 || ti != 2 || c != 1 {
		t.Errorf("(c) nada novo: %d %d %d", i, ti, c)
	}
}

func TestMonitorDoisGraus(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, true, "k")
	a.adv.criarProcesso(t, numA)
	dj.poe(numA, hitG("G1", mov(26, "Distribuição", "2025-03-10T10:00:00.000Z"))+","+hitG("G2", mov(200, "Recebimento", "2025-06-01T00:00:00.000Z")))
	ctx := context.Background()
	_, _ = a.svc.ColetarAgora(ctx, a.ac)
	dj.poe(numA, hitG("G1", mov(26, "Distribuição", "2025-03-10T10:00:00.000Z"), mov(11, "Juntada", "2026-11-03T10:00:00.000Z"))+","+
		hitG("G2", mov(200, "Recebimento", "2025-06-01T00:00:00.000Z"), mov(12, "Julgamento", "2026-11-03T11:00:00.000Z")))
	_, _ = a.svc.ColetarAgora(ctx, a.ac)
	if i, _, _ := a.automaticas(); i != 2 || a.count(`SELECT count(*) FROM processo_movimentos WHERE grau='G2' AND NOT base`) != 1 ||
		a.count(`SELECT count(*) FROM processo_movimentos WHERE grau='G1' AND base`) != 1 {
		t.Errorf("(d) os dois graus devem ser lidos (regressão do Hits[0])")
	}
}

func TestMonitorFalhaNaTarefaDesfazEReexecucaoRegistra(t *testing.T) {
	dj := novoFalsoDataJud(t)
	var ft *falhaTarefa
	a := ambMonitor(t, dj, true, "k", WithTarefaAutomatica(func(x intimacoes.TarefaAutomatica) intimacoes.TarefaAutomatica {
		ft = &falhaTarefa{TarefaAutomatica: x}
		return ft
	}))
	pid := a.adv.criarProcesso(t, numA)
	ctx := context.Background()
	dj.poe(numA, hitG("G1", mov(26, "Distribuição", "2025-03-10T10:00:00.000Z")))
	_, _ = a.svc.ColetarAgora(ctx, a.ac)
	dj.poe(numA, hitG("G1", mov(26, "Distribuição", "2025-03-10T10:00:00.000Z"), mov(11, "Juntada", "2026-11-03T10:00:00.000Z")))
	var antes *time.Time
	_ = a.pool.QueryRow(ctx, `SELECT ultimo_sucesso FROM monitor_processos WHERE processo_id=$1::uuid`, pid).Scan(&antes)
	ft.falha = true
	if _, err := a.svc.ColetarAgora(ctx, a.ac); err != nil {
		t.Fatal(err)
	}
	if i, _, _ := a.automaticas(); i != 0 || a.count(`SELECT count(*) FROM processo_movimentos WHERE NOT base`) != 0 {
		t.Fatal("(e) falha na tarefa: nada pode ficar gravado (visto sem registro)")
	}
	var depois *time.Time
	var st string
	_ = a.pool.QueryRow(ctx, `SELECT ultimo_sucesso, status FROM monitor_processos WHERE processo_id=$1::uuid`, pid).Scan(&depois, &st)
	if st != "falha" || (antes != nil && depois != nil && !depois.Equal(*antes)) {
		t.Errorf("(e) o último sucesso não avança na falha: status=%s", st)
	}
	ft.falha = false
	_, _ = a.svc.ColetarAgora(ctx, a.ac)
	if i, ti, _ := a.automaticas(); i != 1 || ti != 1 {
		t.Errorf("(e) a reexecução registra: %d %d", i, ti)
	}
}

func TestMonitorSemRegistroEParcialNuncaSemNovidades(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, true, "k")
	pid := a.adv.criarProcesso(t, numA)
	r, err := a.svc.ColetarAgora(context.Background(), a.ac)
	if err != nil {
		t.Fatal(err)
	}
	var st, msg string
	_ = a.pool.QueryRow(context.Background(), `SELECT status, mensagem FROM monitor_processos WHERE processo_id=$1::uuid`, pid).Scan(&st, &msg)
	if st != "parcial" || msg != "Sem registro no DataJud para este número (pode haver segredo de justiça, atraso na publicação ou número incorreto). Confira nas fontes oficiais." {
		t.Errorf("(f) zero hits: %q %q", st, msg)
	}
	if regexpSemNovidades.MatchString(msg + r.Mensagem) {
		t.Errorf("(f) nunca 'sem novidades': %q", r.Mensagem)
	}
	if r.Status == "sucesso" {
		t.Error("(f) coleta com processo sem registro não é sucesso pleno")
	}
}

func TestMonitorSemNumeroValidoNaoConsultaERede(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, true, "k")
	pid := a.adv.criarProcesso(t, "")
	_, _ = a.svc.ColetarAgora(context.Background(), a.ac)
	var st, msg string
	_ = a.pool.QueryRow(context.Background(), `SELECT status, mensagem FROM monitor_processos WHERE processo_id=$1::uuid`, pid).Scan(&st, &msg)
	if st != "bloqueado" || !strings.Contains(msg, "sem número CNJ válido") || dj.n() != 0 {
		t.Errorf("(g) %q %q chamadas=%d", st, msg, dj.n())
	}
}

func TestMonitorDesligadoEChaveAusenteNaoUsamARede(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, false, "k")
	a.adv.criarProcesso(t, numA)
	r, err := a.svc.ColetarAgora(context.Background(), a.ac)
	if err != nil || r.Status != "bloqueado" || r.Mensagem != "Coleta desligada neste ambiente." || dj.n() != 0 {
		t.Errorf("(h) desligada: %+v %v chamadas=%d", r, err, dj.n())
	}
	b := ambMonitor(t, dj, true, "")
	b.adv.criarProcesso(t, numA)
	r, err = b.svc.ColetarAgora(context.Background(), b.ac)
	if err != nil || r.Status != "bloqueado" || r.Mensagem != "Chave do DataJud não configurada." || dj.n() != 0 {
		t.Errorf("(h) sem chave: %+v %v chamadas=%d", r, err, dj.n())
	}
}

func TestMonitorDuasColetasSimultaneas(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, true, "k")
	a.adv.criarProcesso(t, numA)
	dj.trava, dj.entrou = make(chan struct{}), make(chan struct{}, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); _, _ = a.svc.ColetarAgora(context.Background(), a.ac) }()
	<-dj.entrou
	_, err := a.svc.ColetarAgora(context.Background(), a.ac)
	close(dj.trava)
	wg.Wait()
	if err == nil || !strings.Contains(err.Error(), "em andamento") {
		t.Errorf("(i) a segunda coleta simultânea deveria ser recusada: %v", err)
	}
}

// E1: TODOS os estados com número válido são consultados; excluído, não.
func TestMonitorConsultaTodosOsEstadosMenosExcluido(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, true, "k")
	p1 := a.adv.criarProcesso(t, numA)
	p2 := a.adv.criarProcesso(t, numB)
	p3 := a.adv.criarProcesso(t, "0001234-25.2023.4.03.6100")
	p4 := a.adv.criarProcesso(t, "0000001-30.2026.6.26.0005")
	ctx := context.Background()
	for p, s := range map[string]string{p2: "extinto", p3: "arquivado"} {
		if _, err := a.pool.Exec(ctx, `UPDATE processos SET situacao=$2 WHERE id=$1::uuid`, p, s); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = a.pool.Exec(ctx, `UPDATE processos SET excluido_em=now() WHERE id=$1::uuid`, p4)
	_, _ = a.svc.ColetarAgora(ctx, a.ac)
	if a.count(`SELECT count(*) FROM monitor_processos WHERE processo_id = ANY($1::uuid[])`, []string{p1, p2, p3}) != 3 ||
		a.count(`SELECT count(*) FROM monitor_processos WHERE processo_id=$1::uuid`, p4) != 0 || dj.n() != 3 {
		t.Errorf("(j/E1) ativo, extinto e arquivado consultados (3 chamadas), excluído não: chamadas=%d", dj.n())
	}
}

func TestMonitorSegredoDeJusticaNoDataJudNaoViraMovimento(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, true, "k")
	pid := a.adv.criarProcesso(t, numA)
	dj.poe(numA, `{"_source":{"tribunal":"TJSP","grau":"G1","nivelSigilo":5,"movimentos":[]}}`)
	_, _ = a.svc.ColetarAgora(context.Background(), a.ac)
	var st, msg string
	_ = a.pool.QueryRow(context.Background(), `SELECT status, mensagem FROM monitor_processos WHERE processo_id=$1::uuid`, pid).Scan(&st, &msg)
	if st != "parcial" || !strings.Contains(msg, "segredo de justiça") {
		t.Errorf("sigilo no DataJud: %q %q", st, msg)
	}
}

func TestMonitorExecutarSoAdministradorEAudita(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, true, "k")
	a.adv.criarProcesso(t, numA)
	if r, _ := a.adv.post("/coleta-djen/monitor/executar", nil); r.StatusCode != 403 {
		t.Errorf("(l) advogado não executa o monitor: %d", r.StatusCode)
	}
	adm, _ := a.user("adm2-mon@exemplo.test", auth.Administrador)
	if r, _ := adm.post("/coleta-djen/monitor/executar", nil); r.StatusCode != 200 {
		t.Errorf("(l) administrador executa: %d", r.StatusCode)
	}
	if a.count(`SELECT count(*) FROM audit_events WHERE action IN ('monitor.coleta_pedida','monitor.coleta')`) < 2 {
		t.Error("(l) pedido e resultado auditados")
	}
}

// falsoSigiloso é o DJEN por processo: registra o filtro por número que recebeu.
type falsoSigiloso struct {
	mu      sync.Mutex
	numeros []string
}

func (f *falsoSigiloso) ColetarPorNumero(_ context.Context, oab, uf, numero string, _, _ time.Time) ([]intimacoes.ComunicacaoDoProcesso, error) {
	f.mu.Lock()
	f.numeros = append(f.numeros, oab+"/"+uf+"|"+numero)
	f.mu.Unlock()
	if numero == "" {
		return nil, fmt.Errorf("sem filtro por número: a pauta inteira nunca é consultada")
	}
	return []intimacoes.ComunicacaoDoProcesso{{ID: "sig-1", Tribunal: "TJSP", Orgao: "2ª Vara", Processo: numero, Tipo: "Intimação",
		Data: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC), Texto: "Texto integral sigiloso (fictício).", Link: "https://exemplo.test/s/1"}}, nil
}

func TestMonitorSigilosoUsaDJENFiltradoPorNumero(t *testing.T) {
	dj := novoFalsoDataJud(t)
	fs := &falsoSigiloso{}
	a := ambMonitor(t, dj, true, "k", WithColetorSigiloso(fs))
	pid := a.adv.criarProcesso(t, numB)
	ctx := context.Background()
	oab, err := a.env.svc.Intimacoes.CadastrarOAB(ctx, a.ac, intimacoes.OABIn{Nome: "Adv Sig", Numero: "654321", UF: "SP",
		AutorizacaoBase: intimacoes.BaseAutorizacaoPadrao, AutorizacaoEm: "2026-11-01", Ativa: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.svc.CadastrarSigiloso(ctx, a.ac, "00000000-0000-0000-0000-000000000000", oab); err == nil {
		t.Error("(k) sigiloso exige processo existente")
	}
	if err := a.svc.CadastrarSigiloso(ctx, a.ac, pid, oab); err != nil {
		t.Fatal(err)
	}
	_, _ = a.svc.ColetarAgora(ctx, a.ac)
	if len(fs.numeros) != 1 || !strings.HasSuffix(fs.numeros[0], "|"+numB) {
		t.Errorf("(k) consulta com filtro por número: %v", fs.numeros)
	}
	if a.count(`SELECT count(*) FROM intimacoes WHERE origem='DJEN (sigiloso)' AND processo_id=$1::uuid AND texto <> ''`, pid) != 1 ||
		a.count(`SELECT count(*) FROM tarefas WHERE intimacao_id IS NOT NULL AND prazo_interno`) != 1 {
		t.Error("(k) o resultado vira linha 'DJEN (sigiloso)' com tarefa de análise e prazo interno")
	}
	_, _ = a.svc.ColetarAgora(ctx, a.ac)
	if a.count(`SELECT count(*) FROM intimacoes WHERE origem='DJEN (sigiloso)'`) != 1 {
		t.Error("(k) repetir não duplica")
	}
}

var regexpSemNovidades = regexp.MustCompile(`(?i)sem novidades|nenhuma novidade|nenhuma movimenta`)
