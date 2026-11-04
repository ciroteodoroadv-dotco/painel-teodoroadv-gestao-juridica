package tarefas

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/pessoas"
	"painelteodoro/internal/platform/migrate"
	"painelteodoro/internal/platform/webx"
)

var chave = []byte("0123456789abcdef0123456789abcdef")

type falsoVinculos struct{ ate, pro map[string][2]string }

func (f falsoVinculos) tab(tipo string) map[string][2]string {
	if tipo == "atendimento" {
		return f.ate
	}
	return f.pro
}
func (f falsoVinculos) Resumos(_ context.Context, tipo string, ids []string) (map[string][2]string, error) {
	out := map[string][2]string{}
	for _, id := range ids {
		if v, ok := f.tab(tipo)[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}
func (f falsoVinculos) Clientes(_ context.Context, tipo string, ids []string) (map[string][]Ref, error) {
	out := map[string][]Ref{}
	for _, id := range ids {
		if v, ok := f.tab(tipo)[id]; ok {
			out[id] = []Ref{{ID: "p-" + id, Codigo: "PES. 0000009", Nome: v[1]}}
		}
	}
	return out, nil
}
func (f falsoVinculos) Opcoes(_ context.Context, tipo string) ([]Opcao, error) {
	var out []Opcao
	for id, v := range f.tab(tipo) {
		out = append(out, Opcao{Valor: id, Rotulo: v[0] + " · " + v[1]})
	}
	return out, nil
}

type chamada struct {
	op, tarefa, motivo string
	data               *time.Time
	confirmar          bool
}
type falsoPrazos struct {
	calls []chamada
	sit   map[string]string
}

func (f *falsoPrazos) Sincronizar(_ context.Context, _ webx.Actor, tarefaID, _, _ string, data *time.Time, confirmar bool) (string, error) {
	f.calls = append(f.calls, chamada{"sincronizar", tarefaID, "", data, confirmar})
	if f.sit == nil {
		f.sit = map[string]string{}
	}
	f.sit[tarefaID] = map[bool]string{true: "confirmado", false: "a_conferir"}[confirmar]
	return f.sit[tarefaID], nil
}
func (f *falsoPrazos) Situacoes(_ context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range ids {
		if s, ok := f.sit[id]; ok {
			out[id] = s
		}
	}
	return out, nil
}
func (f *falsoPrazos) Encerrar(_ context.Context, _ webx.Actor, tarefaID, motivo string) error {
	f.calls = append(f.calls, chamada{"encerrar", tarefaID, motivo, nil, false})
	return nil
}
func (f *falsoPrazos) Confirmar(_ context.Context, _ webx.Actor, tarefaID string) error {
	f.calls = append(f.calls, chamada{"confirmar", tarefaID, "", nil, true})
	return nil
}
func (f *falsoPrazos) ultima() chamada { return f.calls[len(f.calls)-1] }

type falsoDocs struct{ nomes []string }

func (f *falsoDocs) Anexar(_ context.Context, _ webx.Actor, _, _, _, nome string, c io.Reader) error {
	b, _ := io.ReadAll(c)
	f.nomes = append(f.nomes, nome+":"+string(b))
	return nil
}

type amb struct {
	s      *Service
	ac     webx.Actor
	pool   *pgxpool.Pool
	p      map[string]string // pessoas
	ate    string            // id do atendimento fictício
	pro    string            // id do processo fictício
	prazos *falsoPrazos
	docs   *falsoDocs
}

func novoAmb(t *testing.T) *amb {
	t.Helper()
	dsn := os.Getenv("PAINEL_TEST_DB")
	if dsn == "" {
		t.Skip("PAINEL_TEST_DB não definida: teste de integração pulado")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	_, _ = pool.Exec(ctx, `TRUNCATE pessoas, processos CASCADE`)
	if err := migrate.Reset(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(ctx, pool); err != nil {
		t.Fatal(err)
	}
	st := &auth.Store{Pool: pool}
	h, _ := auth.HashPassword("cavalo-bateria-grampo-7")
	tx, _ := pool.Begin(ctx)
	uid, err := st.CreateUser(ctx, tx, "tar@exemplo.test", auth.Advogado, h, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `TRUNCATE pessoas, processos CASCADE`) })
	ac := webx.Actor{UserID: uid, RequestID: "t"}
	pes := &pessoas.Service{Pool: pool, FieldKey: chave}
	ids := map[string]string{}
	for k, nome := range map[string]string{"maria": "Maria Exemplo", "joao": "João Fictício"} {
		id, _, err := pes.Create(ctx, ac, pessoas.Input{TipoPessoa: pessoas.Fisica, NomeCompleto: nome, Nacionalidade: "Brasileiro(a)"})
		if err != nil {
			t.Fatal(err)
		}
		ids[k] = id
	}
	// atendimento e processo fictícios, direto no banco (este pacote não conhece os outros domínios)
	var ate, pro string
	if err := pool.QueryRow(ctx, `INSERT INTO atendimentos (codigo, descricao, data_abertura, created_by, updated_by)
		VALUES ('ATE. 0000001','Demanda','2026-11-03',$1,$1) RETURNING id`, uid).Scan(&ate); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO processos (codigo, numero_cnj, situacao, created_by, updated_by)
		VALUES ('PRO. 0000001','','ativo',$1,$1) RETURNING id`, uid).Scan(&pro); err != nil {
		t.Fatal(err)
	}
	pz, dc := &falsoPrazos{}, &falsoDocs{}
	s := &Service{Pool: pool, Pessoas: pes, Prazos: pz, Docs: dc,
		Vinculos: falsoVinculos{ate: map[string][2]string{ate: {"ATE. 0000001", "MARIA EXEMPLO"}}, pro: map[string][2]string{pro: {"PRO. 0000001", "JOÃO FICTÍCIO"}}}}
	return &amb{s: s, ac: ac, pool: pool, p: ids, ate: ate, pro: pro, prazos: pz, docs: dc}
}

func (a *amb) in(titulo string) Input {
	return Input{OrigemTipo: "atendimento", OrigemID: a.ate, Titulo: titulo, Estado: "pendente", Prioridade: "Normal"}
}

func (a *amb) criar(t *testing.T, in Input, adv bool) string {
	t.Helper()
	id, _, err := a.s.Create(context.Background(), a.ac, in, adv)
	if err != nil {
		t.Fatalf("criar %q: %v", in.Titulo, err)
	}
	return id
}

func invalido(t *testing.T, err error, msg string) {
	t.Helper()
	var inv webx.Invalid
	if !errors.As(err, &inv) || inv.Msg != msg {
		t.Errorf("erro = %v, esperado %q", err, msg)
	}
}

func TestPrioridadeClasse(t *testing.T) {
	for p, want := range map[string]string{"Urgente": "prioridade-alta", "Alta": "prioridade-media", "Normal": "prioridade-normal", "Baixa": "prioridade-baixa", "": "prioridade-normal", "—": "prioridade-normal"} {
		if got := PrioridadeClasse(p); got != want {
			t.Errorf("PrioridadeClasse(%q) = %q, esperado %q", p, got, want)
		}
	}
	if len(Estados) != 7 || Estados[0].Chave != "a-confirmar" || Estados[4].Rotulo != "Concluída com sucesso" || Estados[6].Chave != "canceladas" {
		t.Errorf("estados fora da ordem do B: %+v", Estados)
	}
}

func TestCriarVinculadaAtendimentoEProcesso(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	in := a.in("Protocolar embargos")
	in.Subtitulo, in.Descricao, in.Observacoes, in.Prioridade = "Ligação", "Detalhe", "Obs", "Urgente"
	in.DataBase, in.Prazo = "2026-11-03", "2026-11-10"
	in.CriadoPorID = a.p["joao"]
	in.Partes = []ParteIn{{PessoaID: a.p["maria"], Papel: "Autor"}, {PessoaID: a.p["joao"], Papel: "Réu"}}
	in.Responsaveis = []string{a.p["joao"], a.p["maria"]}
	id, cod, err := a.s.Create(ctx, a.ac, in, true)
	if err != nil || cod != "TAR. 0000001" {
		t.Fatalf("criar = %q %v", cod, err)
	}
	t1, err := a.s.Get(ctx, id)
	if err != nil || t1.OrigemTipo != "atendimento" || t1.OrigemCodigo != "ATE. 0000001" || t1.ClienteNome != "MARIA EXEMPLO" || t1.Estado != "pendente" ||
		t1.Prioridade != "Urgente" || t1.Subtitulo != "Ligação" || t1.CriadoPor != "JOÃO FICTÍCIO" || len(t1.Partes) != 2 || t1.Partes[1].Papel != "Réu" ||
		len(t1.Responsaveis) != 2 || t1.Responsaveis[0].Nome != "JOÃO FICTÍCIO" || t1.Prazo == nil || t1.Prazo.Format("2006-01-02") != "2026-11-10" || t1.PrazoFatal != nil {
		t.Fatalf("leitura = %+v %v", t1, err)
	}
	in2 := Input{OrigemTipo: "processo", OrigemID: a.pro, Titulo: "Juntar procuração", Estado: "em-execucao", Prioridade: "Baixa"}
	id2, cod2, err := a.s.Create(ctx, a.ac, in2, false)
	if err != nil || cod2 != "TAR. 0000002" {
		t.Fatalf("criar vinculada a processo = %q %v", cod2, err)
	}
	t2, _ := a.s.Get(ctx, id2)
	if t2.OrigemCodigo != "PRO. 0000001" || t2.ClienteNome != "JOÃO FICTÍCIO" || t2.Estado != "em-execucao" {
		t.Errorf("tarefa do processo = %+v", t2)
	}
}

func TestValidacoesDaTarefa(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	mut := func(f func(*Input)) Input { in := a.in("Título"); f(&in); return in }
	casos := []struct {
		nome string
		in   Input
		msg  string
	}{
		{"sem título", mut(func(i *Input) { i.Titulo = "  " }), "Título é obrigatório."},
		{"sem tipo de vínculo", mut(func(i *Input) { i.OrigemTipo = "" }), "Selecione o vínculo (Atendimento ou Processo)."},
		{"sem vínculo específico", mut(func(i *Input) { i.OrigemID = "" }), "Selecione o vínculo específico."},
		{"vínculo inexistente", mut(func(i *Input) { i.OrigemID = "00000000-0000-0000-0000-000000000000" }), "Vínculo não encontrado."},
		{"vínculo de outro tipo", mut(func(i *Input) { i.OrigemTipo = "processo" }), "Vínculo não encontrado."},
		{"estado inválido", mut(func(i *Input) { i.Estado = "feita" }), "Estado inválido."},
		{"prioridade inválida", mut(func(i *Input) { i.Prioridade = "Média" }), "Prioridade inválida."},
		{"subtítulo longo", mut(func(i *Input) { i.Subtitulo = strings.Repeat("a", 81) }), "Subtítulo com mais de 80 caracteres."},
		{"data inválida", mut(func(i *Input) { i.PrazoFatal = "10/11/2026" }), "Data inválida em Prazo fatal."},
		{"parte inexistente", mut(func(i *Input) { i.Partes = []ParteIn{{PessoaID: "00000000-0000-0000-0000-000000000000"}} }), "Pessoa não encontrada: cadastre-a em Pessoas antes de usá-la na tarefa."},
		{"pai inexistente", mut(func(i *Input) { i.TarefaPai = "00000000-0000-0000-0000-000000000000" }), "Tarefa de origem não encontrada."},
	}
	for _, c := range casos {
		_, _, err := a.s.Create(ctx, a.ac, c.in, true)
		invalido(t, err, c.msg)
	}
	var n int
	_ = a.pool.QueryRow(ctx, `SELECT count(*) FROM tarefas`).Scan(&n)
	if n != 0 {
		t.Errorf("tarefas gravadas apesar dos erros: %d", n)
	}
}

func TestPrazoFatalSincronizaComODominioPrazos(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	in := a.in("Com prazo fatal")
	in.PrazoFatal = "2026-12-01"
	id := a.criar(t, in, true) // advogado
	if c := a.prazos.ultima(); c.op != "sincronizar" || !c.confirmar || c.data.Format("2006-01-02") != "2026-12-01" || c.tarefa != id {
		t.Fatalf("advogado deveria sincronizar confirmando: %+v", c)
	}
	t1, _ := a.s.Get(ctx, id)
	if t1.PrazoFatalSituacao != "confirmado" {
		t.Errorf("situação = %q", t1.PrazoFatalSituacao)
	}
	in2 := a.in("Do assistente")
	in2.PrazoFatal = "2026-12-02"
	id2 := a.criar(t, in2, false) // não advogado
	if c := a.prazos.ultima(); c.confirmar {
		t.Errorf("não advogado não pode confirmar: %+v", c)
	}
	if t2, _ := a.s.Get(ctx, id2); t2.PrazoFatalSituacao != "a_conferir" {
		t.Errorf("situação do assistente = %q", t2.PrazoFatalSituacao)
	}
	// Editar: sincroniza de novo (o domínio Prazos decide não duplicar); apagar a data cancela o prazo.
	ed := in2
	ed.Titulo = "Do assistente (editada)"
	if err := a.s.Update(ctx, a.ac, id2, 1, ed, false); err != nil {
		t.Fatal(err)
	}
	if c := a.prazos.ultima(); c.op != "sincronizar" || c.tarefa != id2 {
		t.Errorf("editar deveria sincronizar: %+v", c)
	}
	ed.PrazoFatal = ""
	if err := a.s.Update(ctx, a.ac, id2, 2, ed, false); err != nil {
		t.Fatal(err)
	}
	if c := a.prazos.ultima(); c.op != "encerrar" || c.motivo != "cancelada" {
		t.Errorf("apagar a data deveria cancelar o prazo: %+v", c)
	}
}

func TestMoverEstados(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	in := a.in("Automática")
	in.Estado = "a-confirmar"
	id := a.criar(t, in, false)
	// Só advogado tira de "A confirmar".
	invalido(t, a.s.Mover(ctx, a.ac, id, 1, "pendente", false), "Somente advogado confirma tarefas a confirmar.")
	if err := a.s.Mover(ctx, a.ac, id, 1, "pendente", true); err != nil {
		t.Fatalf("advogado movendo: %v", err)
	}
	if t1, _ := a.s.Get(ctx, id); t1.Estado != "pendente" || t1.Version != 2 {
		t.Errorf("depois de mover = %+v", t1)
	}
	if err := a.s.Mover(ctx, a.ac, id, 1, "revisao", true); !errors.Is(err, webx.ErrConflict) {
		t.Errorf("versão antiga = %v", err)
	}
	invalido(t, a.s.Mover(ctx, a.ac, id, 2, "feita", true), "Estado inválido.")
	// Concluir e cancelar encerram o prazo.
	in2 := a.in("Com prazo")
	in2.PrazoFatal = "2026-12-01"
	id2 := a.criar(t, in2, true)
	if err := a.s.Mover(ctx, a.ac, id2, 1, "concluidas-com-sucesso", false); err != nil {
		t.Fatal(err)
	}
	if c := a.prazos.ultima(); c.op != "encerrar" || c.motivo != "cumprida" {
		t.Errorf("concluir deveria cumprir o prazo: %+v", c)
	}
	if err := a.s.Mover(ctx, a.ac, id2, 2, "canceladas", false); err != nil {
		t.Fatal(err)
	}
	if c := a.prazos.ultima(); c.op != "encerrar" || c.motivo != "cancelada" {
		t.Errorf("cancelar deveria cancelar o prazo: %+v", c)
	}
	if err := a.s.Mover(ctx, a.ac, id2, 3, "pendente", false); err != nil { // reabrir: o prazo volta a ser sincronizado
		t.Fatal(err)
	}
	if c := a.prazos.ultima(); c.op != "sincronizar" || c.confirmar {
		t.Errorf("reabrir deveria ressincronizar sem confirmar: %+v", c)
	}
	// Editar o Status no drawer segue a mesma regra de "A confirmar".
	ed := a.in("Automática")
	ed.Estado = "pendente"
	in3 := a.in("Outra automática")
	in3.Estado = "a-confirmar"
	id3 := a.criar(t, in3, false)
	invalido(t, a.s.Update(ctx, a.ac, id3, 1, ed, false), "Somente advogado confirma tarefas a confirmar.")
}

func TestTarefaFilha(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	pai := a.criar(t, a.in("Pai"), true)
	f := a.in("Filha")
	f.TarefaPai = pai
	f.Partes = []ParteIn{{PessoaID: a.p["maria"], Papel: "Autor"}}
	f.Responsaveis = []string{a.p["joao"]}
	id := a.criar(t, f, true)
	t1, _ := a.s.Get(ctx, id)
	if t1.TarefaPaiID != pai || t1.TarefaPaiCodigo != "TAR. 0000001" || len(t1.Partes) != 1 || len(t1.Responsaveis) != 1 {
		t.Errorf("filha = %+v", t1)
	}
}

func TestCancelarNaoApagaEAuditaDePara(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	in := a.in("A cancelar")
	in.PrazoFatal = "2026-12-01"
	id := a.criar(t, in, true)
	antes, _ := a.s.Get(ctx, id)
	if err := a.s.Mover(ctx, a.ac, id, 1, "canceladas", true); err != nil {
		t.Fatal(err)
	}
	dep, err := a.s.Get(ctx, id)
	if err != nil || dep.Estado != "canceladas" || dep.Codigo != antes.Codigo || dep.Titulo != antes.Titulo {
		t.Fatalf("a tarefa cancelada permanece, com o mesmo código e título: %+v %v", dep, err)
	}
	if l, _ := a.s.Listar(ctx, ""); len(l) != 1 {
		t.Errorf("a cancelada continua na lista (%d)", len(l))
	}
	if c := a.prazos.ultima(); c.op != "encerrar" || c.motivo != "cancelada" {
		t.Errorf("cancelar encerra o prazo ligado: %+v", c)
	}
	var meta string
	if err := a.pool.QueryRow(ctx, `SELECT meta::text FROM audit_events WHERE action='tarefa.estado_alterado' AND target_id=$1`, id).Scan(&meta); err != nil {
		t.Fatalf("auditoria tarefa.estado_alterado: %v", err)
	}
	if !strings.Contains(meta, "pendente") || !strings.Contains(meta, "canceladas") {
		t.Errorf("a auditoria guarda de→para: %s", meta)
	}
	var apagadas int
	_ = a.pool.QueryRow(ctx, `SELECT count(*) FROM tarefas WHERE excluido_em IS NOT NULL`).Scan(&apagadas)
	if apagadas != 0 {
		t.Error("nenhuma tarefa é apagada")
	}
}

func TestBuscaSemAcento(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	in := a.in("Protocolar petição")
	in.Responsaveis = []string{a.p["joao"]}
	in.Descricao = "Atualização do contrato"
	a.criar(t, in, true)
	in2 := Input{OrigemTipo: "processo", OrigemID: a.pro, Titulo: "Outra coisa", Estado: "pendente", Prioridade: "Normal"}
	a.criar(t, in2, true)
	for q, want := range map[string]int{"": 2, "peticao": 1, "MARIA": 1, "tar. 0000002": 1, "ficticio": 2, "ate. 0000001": 1, "pro. 0000001": 1, "atualizacao": 1, "nada": 0} {
		l, err := a.s.Listar(ctx, q)
		if err != nil || len(l) != want {
			t.Errorf("busca %q = %d (%v), esperado %d", q, len(l), err, want)
		}
	}
}

func TestAgendaTarefas(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	hoje := webx.Hoje(time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC))
	mk := func(titulo, sub, estado, prazo string) string {
		in := a.in(titulo)
		in.Subtitulo, in.Estado, in.Prazo = sub, estado, prazo
		return a.criar(t, in, true)
	}
	mk("L1", "Ligação", "pendente", "2026-11-10")
	mk("L2", "ligacoes", "revisao", "2026-11-05")
	mk("L3", "Ligação", "concluidas-com-sucesso", "")
	mk("L4", "Ligação", "canceladas", "")
	mk("R1", "Reunião", "em-execucao", "2026-11-03")
	mk("R2", "REUNIOES", "a-confirmar", "")
	mk("A1", "Aula", "pendente", "2026-10-01")
	mk("A2", "aulas", "pendente", "")
	excl := mk("L5", "Ligação", "pendente", "")
	if err := a.s.Mover(ctx, a.ac, excl, 1, "canceladas", true); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		norm []string
		want int
	}{{[]string{"ligacao", "ligacoes"}, 2}, {[]string{"reuniao", "reunioes"}, 2}, {[]string{"aula", "aulas"}, 2}, {[]string{"monitoramento"}, 0}} {
		l, err := a.s.AgendaPorSubtitulo(ctx, hoje, c.norm)
		if err != nil || len(l) != c.want {
			t.Errorf("AgendaPorSubtitulo(%v) = %d (%v), esperado %d", c.norm, len(l), err, c.want)
		}
	}
	// Ordem por prazo (L2 antes de L1) e dados da linha: cliente com PES, selo e prazo.
	l, _ := a.s.AgendaPorSubtitulo(ctx, hoje, []string{"ligacao", "ligacoes"})
	if l[0].Titulo != "L2" || l[0].Prazo != "05/11/2026" || l[0].PrazoFatal != "—" || len(l[0].Clientes) != 1 || l[0].Clientes[0].Rotulo() != "MARIA EXEMPLO · PES. 0000009" {
		t.Errorf("linha da agenda = %+v", l[0])
	}
	if l[0].StatusRotulo != "Revisão" || l[0].StatusClasse != "badge-neutral" {
		t.Errorf("selo = %q %q", l[0].StatusRotulo, l[0].StatusClasse)
	}
	// Atrasada: prazo anterior a hoje e ainda aberta.
	al, _ := a.s.AgendaPorSubtitulo(ctx, hoje, []string{"aula", "aulas"})
	var atrasada AgendaLinha
	for _, x := range al {
		if x.Titulo == "A1" {
			atrasada = x
		}
	}
	if atrasada.StatusRotulo != "Pendente · atrasada" || atrasada.StatusClasse != "badge-danger" {
		t.Errorf("atrasada = %q %q", atrasada.StatusRotulo, atrasada.StatusClasse)
	}
	// Categoria Tarefas = todas em aberto (L1, L2, R1, R2, A1, A2).
	if tl, err := a.s.AgendaEmAberto(ctx, hoje); err != nil || len(tl) != 6 {
		t.Errorf("AgendaEmAberto = %d (%v), esperado 6", len(tl), err)
	}
	// Aviso de mesmo dia: só tarefas abertas com esse prazo.
	if ts, _ := a.s.TitulosComPrazoEm(ctx, hoje); len(ts) != 1 || ts[0] != "R1" {
		t.Errorf("TitulosComPrazoEm = %v", ts)
	}
}

func TestAbertosDoProcessoEDocumentos(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	in := Input{OrigemTipo: "processo", OrigemID: a.pro, Titulo: "No processo", Estado: "pendente", Prioridade: "Normal"}
	id := a.criar(t, in, true)
	in.Titulo, in.Estado = "Concluída", "concluidas-com-sucesso"
	a.criar(t, in, true)
	if n, err := a.s.AbertosDoProcesso(ctx, a.pro); err != nil || n != 1 {
		t.Errorf("abertos do processo = %d %v", n, err)
	}
	if err := a.s.AnexarDocumento(ctx, a.ac, id, "peticao.pdf", strings.NewReader("%PDF-1.4")); err != nil {
		t.Fatal(err)
	}
	if len(a.docs.nomes) != 1 || a.docs.nomes[0] != "peticao.pdf:%PDF-1.4" {
		t.Errorf("documentos enviados = %v", a.docs.nomes)
	}
	invalido(t, a.s.AnexarDocumento(ctx, a.ac, id, "foto.png", strings.NewReader("x")), "Envie um arquivo PDF.")
	if err := a.s.AnexarDocumento(ctx, a.ac, "00000000-0000-0000-0000-000000000000", "a.pdf", strings.NewReader("x")); !errors.Is(err, webx.ErrNotFound) {
		t.Errorf("tarefa inexistente = %v", err)
	}
}
