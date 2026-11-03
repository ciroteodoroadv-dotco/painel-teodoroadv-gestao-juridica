package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/intimacoes"
	"painelteodoro/internal/intimacoes/djen"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
)

// falsoDJEN é o DJEN dos testes: devolve o que se mandar, conta as chamadas e pode travar para provar a concorrência.
type falsoDJEN struct {
	mu       sync.Mutex
	chamadas []string
	itens    []djen.Comunicacao
	err      error
	parcial  bool
	trava    chan struct{} // se não nulo, Coletar espera aqui
	entrou   chan struct{}
}

func (f *falsoDJEN) Coletar(_ context.Context, oab, uf string, _, _ time.Time) (djen.Resultado, error) {
	f.mu.Lock()
	f.chamadas = append(f.chamadas, oab+"/"+uf)
	itens, err, parcial, trava, entrou := f.itens, f.err, f.parcial, f.trava, f.entrou
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
	if err != nil {
		return djen.Resultado{}, err
	}
	return djen.Resultado{Itens: itens, Parcial: parcial}, nil
}

func (f *falsoDJEN) n() int { f.mu.Lock(); defer f.mu.Unlock(); return len(f.chamadas) }

func comunicacao(id, processo string) djen.Comunicacao {
	return djen.Comunicacao{ID: id, Tribunal: "TJXX", Orgao: "1ª Vara de Exemplo", Processo: processo, Tipo: "Intimação",
		DataPublicacao: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), Texto: "Fica intimada a parte (exemplo).", Link: "https://exemplo.test/c/" + id}
}

// ambDJEN monta o ambiente com coleta ATIVA e o DJEN falso, um administrador e uma OAB autorizada.
func ambDJEN(t *testing.T, f *falsoDJEN, opts ...Option) (*env, webx.Actor, string) {
	t.Helper()
	agora := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	opts = append([]Option{WithDJEN(DJEN{Ativo: true, Transport: nil, AlertaHoras: 6}), WithColetor(f), WithRelogio(func() time.Time { return agora })}, opts...)
	e := newEnvOpts(t, opts...)
	_, uid := e.user("adm-djen@exemplo.test", auth.Administrador)
	ac := webx.Actor{UserID: uid, RequestID: "t"}
	oab, err := e.svc.Intimacoes.CadastrarOAB(context.Background(), ac, intimacoes.OABIn{Nome: "Advogado Fictício", Numero: "123456", UF: "SP",
		AutorizacaoBase: intimacoes.BaseAutorizacaoPadrao, AutorizacaoEm: "2026-11-01", Ativa: true})
	if err != nil {
		t.Fatal(err)
	}
	return e, ac, oab
}

func (e *env) tarefasDeIntimacao() int {
	return e.count(`SELECT count(*) FROM tarefas WHERE intimacao_id IS NOT NULL`)
}

func TestColetaCriaIntimacaoETarefaAConfirmarSemPrazo(t *testing.T) {
	f := &falsoDJEN{itens: []djen.Comunicacao{comunicacao("1", "0000101-25.2025.8.26.0100"), comunicacao("2", "")}}
	e, ac, oab := ambDJEN(t, f)
	r, err := e.svc.Intimacoes.ColetarAgora(context.Background(), ac)
	if err != nil || r.Status != "sucesso" || r.Novas != 2 {
		t.Fatalf("coleta = %+v err=%v", r, err)
	}
	if e.count(`SELECT count(*) FROM intimacoes WHERE status='triagem' AND oab_id=$1::uuid`, oab) != 2 {
		t.Error("deveria haver 2 intimações em triagem")
	}
	if e.count(`SELECT count(*) FROM tarefas WHERE intimacao_id IS NOT NULL AND estado='a-confirmar' AND prazo_fatal IS NULL AND prazo IS NULL
		AND subtitulo='Análise de decisão' AND prioridade='Normal'`) != 2 {
		t.Error("cada intimação nova deve abrir uma tarefa a-confirmar, sem nenhuma data")
	}
	if e.count(`SELECT count(*) FROM prazos`) != 0 {
		t.Error("(k) a tarefa de análise não pode virar prazo")
	}
	var titulo, desc string
	_ = e.pool.QueryRow(context.Background(), `SELECT titulo, descricao FROM tarefas WHERE intimacao_id IS NOT NULL ORDER BY codigo LIMIT 1`).Scan(&titulo, &desc)
	if !strings.HasPrefix(titulo, "Analisar intimação — Intimação") || !strings.Contains(desc, "Prazo a apurar pelo advogado. O sistema não calculou prazo.") ||
		!strings.Contains(desc, "https://exemplo.test/c/") {
		t.Errorf("título/descrição: %q / %q", titulo, desc)
	}
	est, ok, _ := e.svc.Coletas.Obter(context.Background(), intimacoes.Modulo)
	if !ok || est.Status != "sucesso" || est.UltimoSucesso == nil {
		t.Errorf("estado da coleta: %+v", est)
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action IN ('intimacao.registrada','intimacao.coleta','intimacao.coleta_pedida')`) < 4 {
		t.Error("registro, coleta e pedido de coleta devem ser auditados")
	}
}

func TestSegundaColetaNaoDuplica(t *testing.T) {
	f := &falsoDJEN{itens: []djen.Comunicacao{comunicacao("1", ""), comunicacao("2", "")}}
	e, ac, _ := ambDJEN(t, f)
	ctx := context.Background()
	if _, err := e.svc.Intimacoes.ColetarAgora(ctx, ac); err != nil {
		t.Fatal(err)
	}
	r, err := e.svc.Intimacoes.ColetarAgora(ctx, ac)
	if err != nil || r.Novas != 0 || r.Repetidas != 2 {
		t.Fatalf("segunda coleta = %+v err=%v", r, err)
	}
	if e.count(`SELECT count(*) FROM intimacoes`) != 2 || e.tarefasDeIntimacao() != 2 {
		t.Error("nada pode ser duplicado")
	}
}

// falhaTarefa força o erro na criação da tarefa de análise.
type falhaTarefa struct {
	intimacoes.TarefaAutomatica
	falha bool
}

func (f *falhaTarefa) CriarAnaliseDeIntimacao(ctx context.Context, tx pgx.Tx, in intimacoes.AnaliseIn) (string, error) {
	if f.falha {
		return "", errors.New("falha forçada")
	}
	return f.TarefaAutomatica.CriarAnaliseDeIntimacao(ctx, tx, in)
}

func TestFalhaNaTarefaDesfazTudoEJanelaNaoAvanca(t *testing.T) {
	f := &falsoDJEN{itens: []djen.Comunicacao{comunicacao("1", "")}}
	var ft *falhaTarefa
	e, ac, oab := ambDJEN(t, f, WithTarefaAutomatica(func(t intimacoes.TarefaAutomatica) intimacoes.TarefaAutomatica {
		ft = &falhaTarefa{TarefaAutomatica: t, falha: true}
		return ft
	}))
	ctx := context.Background()
	r, err := e.svc.Intimacoes.ColetarAgora(ctx, ac)
	if err != nil || r.Status != "falha" || r.FalhasItem != 1 {
		t.Fatalf("coleta com falha = %+v err=%v", r, err)
	}
	if e.count(`SELECT count(*) FROM intimacoes`) != 0 {
		t.Fatal("a intimação não pode ficar gravada sem a tarefa (visto sem registro)")
	}
	if e.count(`SELECT count(*) FROM oab_monitoradas WHERE id=$1::uuid AND ultimo_sucesso IS NOT NULL`, oab) != 0 {
		t.Error("a janela não pode avançar")
	}
	est, _, _ := e.svc.Coletas.Obter(ctx, intimacoes.Modulo)
	if est.Status != "falha" || est.UltimoSucesso != nil {
		t.Errorf("estado = %+v", est)
	}
	ft.falha = false // a próxima rodada recolhe o que faltou
	r, err = e.svc.Intimacoes.ColetarAgora(ctx, ac)
	if err != nil || r.Status != "sucesso" || r.Novas != 1 || e.tarefasDeIntimacao() != 1 {
		t.Fatalf("recuperação = %+v err=%v", r, err)
	}
}

func TestProcessoExistenteViraVinculoReal(t *testing.T) {
	num := br.MakeCNJ("0000101", "2025", "8", "26", "0100")
	f := &falsoDJEN{itens: []djen.Comunicacao{comunicacao("1", num), comunicacao("2", br.MakeCNJ("0000999", "2025", "8", "26", "0100"))}}
	e, ac, _ := ambDJEN(t, f)
	adv, _ := e.user("adv-djen@exemplo.test", auth.Advogado)
	procID := adv.criarProcesso(t, num)
	if _, err := e.svc.Intimacoes.ColetarAgora(context.Background(), ac); err != nil {
		t.Fatal(err)
	}
	if e.count(`SELECT count(*) FROM tarefas t JOIN intimacoes i ON i.id=t.intimacao_id WHERE t.origem_tipo='processo' AND t.processo_id=$1::uuid AND i.processo_id=$1::uuid`, procID) != 1 {
		t.Error("número existente deve virar vínculo com o processo do v2")
	}
	if e.count(`SELECT count(*) FROM tarefas WHERE origem_tipo='intimacao' AND processo_id IS NULL AND intimacao_id IS NOT NULL`) != 1 {
		t.Error("número inexistente deve gerar tarefa com origem intimacao")
	}
}

func TestOABSemAutorizacaoERecusada(t *testing.T) {
	f := &falsoDJEN{}
	e, ac, _ := ambDJEN(t, f)
	ctx := context.Background()
	svc := e.svc.Intimacoes
	base := intimacoes.OABIn{Nome: "Outro Advogado", Numero: "777", UF: "RJ", AutorizacaoBase: intimacoes.BaseAutorizacaoPadrao, AutorizacaoEm: "2026-11-01", Ativa: true}
	semBase, semData, futura := base, base, base
	semBase.AutorizacaoBase = "  "
	semData.AutorizacaoEm = ""
	futura.AutorizacaoEm = "2026-12-31"
	for nome, in := range map[string]intimacoes.OABIn{"sem base": semBase, "sem data": semData, "data futura": futura} {
		if _, err := svc.CadastrarOAB(ctx, ac, in); err == nil {
			t.Errorf("%s: cadastro deveria ser recusado", nome)
		}
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='oab.cadastro_recusado' AND result='negado'`) != 3 {
		t.Error("as 3 recusas devem ser auditadas")
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='oab.cadastrada' AND actor=$1`, ac.UserID) != 1 {
		t.Error("o cadastro deve auditar quem cadastrou")
	}
	if _, err := svc.CadastrarOAB(ctx, ac, base); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CadastrarOAB(ctx, ac, base); err == nil {
		t.Error("OAB repetida deveria ser recusada")
	}
	// Uma OAB com autorização ainda por vir (gravada direto no banco) é recusada na coleta e a recusa é auditada.
	if _, err := e.pool.Exec(ctx, `UPDATE oab_monitoradas SET autorizacao_em='2026-12-31' WHERE numero_oab='777'`); err != nil {
		t.Fatal(err)
	}
	f.itens = []djen.Comunicacao{comunicacao("1", "")}
	r, err := svc.ColetarAgora(ctx, ac)
	if err != nil || r.Recusadas != 1 || r.Status != "parcial" {
		t.Fatalf("coleta = %+v err=%v", r, err)
	}
	if f.n() != 1 || f.chamadas[0] != "123456/SP" {
		t.Errorf("a OAB sem autorização vigente não pode ser consultada: %v", f.chamadas)
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='intimacao.coleta_recusada'`) != 1 {
		t.Error("a recusa da coleta deve ser auditada")
	}
}

// semRedeFalha prova que, desligada, a coleta não toca a rede: o Coletor real tem um transporte que reprova o teste.
func TestColetaDesligadaNaoFazRede(t *testing.T) {
	e := newEnvOpts(t, WithDJEN(DJEN{Ativo: false, Transport: semRede{t}}))
	_, uid := e.user("adm-off@exemplo.test", auth.Administrador)
	ac := webx.Actor{UserID: uid, RequestID: "t"}
	if _, err := e.svc.Intimacoes.CadastrarOAB(context.Background(), ac, intimacoes.OABIn{Nome: "Advogado Fictício", Numero: "123456", UF: "SP",
		AutorizacaoBase: intimacoes.BaseAutorizacaoPadrao, AutorizacaoEm: "2026-11-01", Ativa: true}); err != nil {
		t.Fatal(err)
	}
	r, err := e.svc.Intimacoes.ColetarAgora(context.Background(), ac)
	if err != nil || r.Status != "bloqueado" || r.Mensagem != "Coleta desligada neste ambiente." {
		t.Fatalf("coleta desligada = %+v err=%v", r, err)
	}
	est, _, _ := e.svc.Coletas.Obter(context.Background(), intimacoes.Modulo)
	if est.Status != "bloqueado" || est.UltimoSucesso != nil {
		t.Errorf("estado = %+v", est)
	}
	if e.svc.Intimacoes.Iniciar(context.Background(), nil, time.Millisecond) {
		t.Error("o agendador não pode nascer com a coleta desligada")
	}
	if alerta, _, ativo, _ := e.svc.Intimacoes.Alerta(context.Background()); ativo || alerta != "" {
		t.Error("desligada, não há faixa de alerta")
	}
}

func TestDuasColetasSimultaneasSoUmaExecuta(t *testing.T) {
	f := &falsoDJEN{itens: []djen.Comunicacao{comunicacao("1", "")}, trava: make(chan struct{}), entrou: make(chan struct{}, 1)}
	e, ac, _ := ambDJEN(t, f)
	ctx := context.Background()
	fim := make(chan error, 1)
	go func() { _, err := e.svc.Intimacoes.ColetarAgora(ctx, ac); fim <- err }()
	<-f.entrou // a primeira já está dentro do DJEN falso, segurando o trinco
	if _, err := e.svc.Intimacoes.ColetarAgora(ctx, ac); !errors.Is(err, intimacoes.ErrOcupada) {
		t.Errorf("a segunda coleta deveria ser recusada como ocupada: %v", err)
	}
	close(f.trava)
	if err := <-fim; err != nil {
		t.Fatal(err)
	}
	if f.n() != 1 || e.tarefasDeIntimacao() != 1 {
		t.Errorf("só uma coleta executa: chamadas=%d tarefas=%d", f.n(), e.tarefasDeIntimacao())
	}
	// Liberado o trinco, uma nova coleta volta a poder rodar.
	f.mu.Lock()
	f.trava = nil
	f.mu.Unlock()
	if _, err := e.svc.Intimacoes.ColetarAgora(ctx, ac); err != nil {
		t.Errorf("depois de terminar, nova coleta: %v", err)
	}
}

func TestHeartbeatAlertaComMaisDeSeisHoras(t *testing.T) {
	e, _, _ := ambDJEN(t, &falsoDJEN{})
	ctx := context.Background()
	agora := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	svc := e.svc.Intimacoes
	if _, _, ativo, _ := svc.Alerta(ctx); !ativo {
		t.Error("coleta ativa sem nenhum sucesso registrado deve alertar")
	}
	reg := func(h float64) {
		t0 := agora.Add(-time.Duration(h * float64(time.Hour)))
		if _, err := e.pool.Exec(ctx, `INSERT INTO coletas (modulo, origem, ultima_tentativa, ultimo_sucesso, status) VALUES ($1,'DJEN',$2,$2,'sucesso')
			ON CONFLICT (modulo) DO UPDATE SET ultima_tentativa=$2, ultimo_sucesso=$2`, intimacoes.Modulo, t0); err != nil {
			t.Fatal(err)
		}
	}
	reg(7)
	forte, texto, ativo, err := svc.Alerta(ctx)
	if err != nil || !ativo || !strings.HasPrefix(forte, "Coleta de intimações sem sucesso desde ") || !strings.Contains(texto, "confira o DJEN manualmente") {
		t.Errorf("último sucesso há 7 h: %q %q %v %v", forte, texto, ativo, err)
	}
	reg(5)
	if _, _, ativo, _ := svc.Alerta(ctx); ativo {
		t.Error("há 5 h ainda não alerta")
	}
}

func TestResumoLongoCortadoPorCaracteresELinkInseguroDescartado(t *testing.T) {
	c := comunicacao("1", "")
	c.Texto = strings.Repeat("€", 3000)
	c.Link = "" // o cliente já descarta http://; o banco recusaria
	f := &falsoDJEN{itens: []djen.Comunicacao{c}}
	e, ac, _ := ambDJEN(t, f)
	if _, err := e.svc.Intimacoes.ColetarAgora(context.Background(), ac); err != nil {
		t.Fatal(err)
	}
	var resumo string
	_ = e.pool.QueryRow(context.Background(), `SELECT resumo FROM intimacoes`).Scan(&resumo)
	if utf8.RuneCountInString(resumo) != 2000 || !utf8.ValidString(resumo) {
		t.Errorf("resumo = %d caracteres", utf8.RuneCountInString(resumo))
	}
}

func TestDecidirConfirmaDescartaEAudita(t *testing.T) {
	f := &falsoDJEN{itens: []djen.Comunicacao{comunicacao("1", ""), comunicacao("2", "")}}
	e, ac, _ := ambDJEN(t, f)
	ctx := context.Background()
	svc := e.svc.Intimacoes
	if _, err := svc.ColetarAgora(ctx, ac); err != nil {
		t.Fatal(err)
	}
	linhas, ok, _, err := svc.Lista(ctx)
	if err != nil || !ok || len(linhas) != 2 {
		t.Fatalf("lista = %d ok=%v err=%v", len(linhas), ok, err)
	}
	if linhas[0].Status != "triagem" || linhas[0].StatusRotulo != "Em triagem" || linhas[0].StatusClasse != "badge-warn" ||
		linhas[0].Tarefa == "" || linhas[0].TarefaEstado != "A confirmar" || linhas[0].Data != "02/11/2026" || linhas[0].Origem != "DJEN" {
		t.Errorf("linha = %+v", linhas[0])
	}
	_, adv := e.user("adv-dec@exemplo.test", auth.Advogado)
	acA := webx.Actor{UserID: adv, RequestID: "t"}
	if err := svc.Decidir(ctx, acA, linhas[0].ID, "confirmada"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Decidir(ctx, acA, linhas[1].ID, "descartada"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Decidir(ctx, acA, linhas[0].ID, "descartada"); err == nil {
		t.Error("intimação já decidida não pode ser decidida de novo")
	}
	if err := svc.Decidir(ctx, acA, linhas[0].ID, "talvez"); err == nil {
		t.Error("decisão inválida")
	}
	linhas, _, _, _ = svc.Lista(ctx)
	if len(linhas) != 2 {
		t.Fatal("a descartada permanece no histórico")
	}
	got := map[string]string{}
	for _, l := range linhas {
		got[l.Status] = l.StatusRotulo + "/" + l.StatusClasse
	}
	if got["confirmada"] != "Confirmada/badge-ok" || got["descartada"] != "Descartada/badge-neutral" {
		t.Errorf("rótulos/classes: %v", got)
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action IN ('intimacao.confirmada','intimacao.descartada') AND actor=$1`, adv) != 2 {
		t.Error("quem decidiu deve ficar na auditoria")
	}
	if e.count(`SELECT count(*) FROM intimacoes WHERE status<>'triagem' AND decidida_por=$1::uuid AND decidida_em IS NOT NULL`, adv) != 2 {
		t.Error("quem/quando devem ser gravados")
	}
}

func TestListaIndisponivelNaoDizNenhumaIntimacao(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	linhas, ok, msg, err := e.svc.Intimacoes.Lista(ctx)
	if err != nil || ok || len(linhas) != 0 || !strings.Contains(msg, "Confira o DJEN manualmente") {
		t.Fatalf("sem coleta nenhuma: ok=%v msg=%q err=%v", ok, msg, err)
	}
	_ = e.svc.Intimacoes.IniciarEstado(ctx)
	_, ok, msg, _ = e.svc.Intimacoes.Lista(ctx)
	if ok || !strings.Contains(msg, "Coleta desligada neste ambiente.") || !strings.Contains(msg, "Confira o DJEN manualmente") {
		t.Errorf("desligada: ok=%v msg=%q", ok, msg)
	}
	// Só depois de uma coleta concluída, lista vazia vira "nenhuma intimação registrada no histórico".
	_, _ = e.pool.Exec(ctx, `UPDATE coletas SET status='sucesso', ultimo_sucesso=now() WHERE modulo=$1`, intimacoes.Modulo)
	if _, ok, _, _ := e.svc.Intimacoes.Lista(ctx); !ok {
		t.Error("após coleta concluída a lista vazia é legítima")
	}
}

func TestAgendadorParaNoContexto(t *testing.T) {
	f := &falsoDJEN{}
	e, ac, _ := ambDJEN(t, f)
	_ = ac
	e.svc.Intimacoes.Cfg.Intervalo = 10 * time.Millisecond
	ctx, cancela := context.WithCancel(context.Background())
	if !e.svc.Intimacoes.Iniciar(ctx, nil, time.Millisecond) {
		t.Fatal("com a coleta ativa o agendador deve iniciar")
	}
	deadline := time.Now().Add(3 * time.Second)
	for f.n() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	cancela()
	if f.n() < 2 {
		t.Fatalf("o agendador deveria ter rodado ao menos 2 vezes, rodou %d", f.n())
	}
	time.Sleep(60 * time.Millisecond) // deixa terminar uma rodada em curso
	apos := f.n()
	time.Sleep(80 * time.Millisecond)
	if f.n() != apos {
		t.Error("o agendador deve parar no ctx.Done()")
	}
}

var _ = http.StatusOK
