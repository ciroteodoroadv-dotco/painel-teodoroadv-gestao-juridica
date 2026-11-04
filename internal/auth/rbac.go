package auth

// Role é o papel do usuário (menor privilégio). Papel desconhecido não tem permissão nenhuma.
type Role string

const (
	Administrador Role = "administrador"
	Advogado      Role = "advogado"
	Assistente    Role = "assistente"
)

// Perm é uma permissão "módulo:ação". Toda rota declara uma — ou é explicitamente pública.
type Perm string

const (
	// Marcadores de acesso (não passam pela matriz de papéis).
	Public      Perm = "publico"         // sem sessão (login, estáticos)
	MFAPending  Perm = "mfa:pendente"    // sessão com senha ok, 2FA ainda não
	AnyLoggedIn Perm = "sessao:qualquer" // sessão completa, qualquer papel
	AnySession  Perm = "sessao:etapa"    // sessão em qualquer etapa (o handler decide o que pode)

	PermHome              Perm = "home:ler"
	PermAuditRead         Perm = "auditoria:ler"
	PermUsersManage       Perm = "usuarios:gerir"
	PermPessoasRead       Perm = "pessoas:ler"
	PermPessoasWrite      Perm = "pessoas:escrever"
	PermProcRead          Perm = "processos:ler"
	PermProcWrite         Perm = "processos:escrever"
	PermTarefaRead        Perm = "tarefas:ler"
	PermTarefaWrite       Perm = "tarefas:escrever"
	PermAtendimentoRead   Perm = "atendimentos:ler"
	PermAtendimentoWrite  Perm = "atendimentos:escrever"
	PermAgendaRead        Perm = "agenda:ler"
	PermAgendaWrite       Perm = "agenda:escrever"
	PermAudienciaRead     Perm = "audiencias:ler"
	PermAudienciaWrite    Perm = "audiencias:escrever"
	PermAudienciaCancel   Perm = "audiencias:cancelar" // cancelar com reautenticação: SÓ advogado
	PermDiligenciaRead    Perm = "diligencias:ler"
	PermDiligenciaWrite   Perm = "diligencias:escrever"
	PermCompromissoCancel Perm = "agenda:cancelar"      // cancelar compromisso (com motivo): SÓ advogado
	PermDiligenciaCancel  Perm = "diligencias:cancelar" // cancelar diligência (com motivo): SÓ advogado
	PermIntimacaoDecide   Perm = "intimacoes:decidir"   // confirmar/descartar intimação: SÓ advogado
	PermColetaManage      Perm = "coleta:gerir"         // OABs da coleta DJEN e "Executar coleta agora": SÓ administrador
	PermDocRead           Perm = "documentos:ler"
	PermDocWrite          Perm = "documentos:escrever"
	PermPrazoRead         Perm = "prazos:ler"
	PermPrazoWrite        Perm = "prazos:escrever"  // registrar/editar (o prazo fica "a conferir")
	PermPrazoConfirm      Perm = "prazos:confirmar" // confirmar, cumprir, cancelar, excluir: SÓ advogado
)

var leitura = []Perm{PermHome, PermPessoasRead, PermProcRead, PermTarefaRead, PermAtendimentoRead, PermAgendaRead, PermAudienciaRead, PermDiligenciaRead, PermDocRead, PermPrazoRead}

// matrix: negar por padrão. Administrador gere usuários e vê auditoria (não mexe em processos);
// advogado lê e escreve nos domínios e é o único que confirma prazo, cancela (audiência, compromisso, diligência, intimação)
// e decide intimação; assistente lê tudo e escreve em tarefas, atendimentos, agenda, audiências, diligências e prazos (que ficam "a conferir").
// Nada se apaga: tarefa, audiência, compromisso, diligência e intimação só se cancelam (determinação de 04/11/2026).
var matrix = func() map[Role]map[Perm]bool {
	m := map[Role]map[Perm]bool{Administrador: {}, Advogado: {}, Assistente: {}}
	for _, p := range leitura {
		m[Administrador][p], m[Advogado][p], m[Assistente][p] = true, true, true
	}
	for _, p := range []Perm{PermUsersManage, PermAuditRead, PermColetaManage} {
		m[Administrador][p] = true
	}
	for _, p := range []Perm{PermPessoasWrite, PermProcWrite, PermTarefaWrite, PermAtendimentoWrite, PermAgendaWrite, PermAudienciaWrite, PermAudienciaCancel, PermDiligenciaWrite, PermCompromissoCancel, PermDiligenciaCancel, PermIntimacaoDecide, PermDocWrite, PermPrazoWrite, PermPrazoConfirm} {
		m[Advogado][p] = true
	}
	for _, p := range []Perm{PermTarefaWrite, PermAtendimentoWrite, PermAgendaWrite, PermAudienciaWrite, PermDiligenciaWrite, PermPrazoWrite} {
		m[Assistente][p] = true
	}
	return m
}()

// Can diz se o papel tem a permissão. Papel ou permissão desconhecidos → false.
func Can(r Role, p Perm) bool { return matrix[r][p] }

// ValidRole informa se o papel existe.
func ValidRole(r Role) bool { _, ok := matrix[r]; return ok }
