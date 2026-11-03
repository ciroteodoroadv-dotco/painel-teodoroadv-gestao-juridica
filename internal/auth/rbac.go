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

	PermHome             Perm = "home:ler"
	PermAuditRead        Perm = "auditoria:ler"
	PermUsersManage      Perm = "usuarios:gerir"
	PermPessoasRead      Perm = "pessoas:ler"
	PermPessoasWrite     Perm = "pessoas:escrever"
	PermProcRead         Perm = "processos:ler"
	PermProcWrite        Perm = "processos:escrever"
	PermTarefaRead       Perm = "tarefas:ler"
	PermTarefaWrite      Perm = "tarefas:escrever"
	PermTarefaDelete     Perm = "tarefas:excluir" // exclusão com reautenticação: advogado e administrador
	PermAtendimentoRead  Perm = "atendimentos:ler"
	PermAtendimentoWrite Perm = "atendimentos:escrever"
	PermAgendaRead       Perm = "agenda:ler"
	PermAgendaWrite      Perm = "agenda:escrever"
	PermDocRead          Perm = "documentos:ler"
	PermDocWrite         Perm = "documentos:escrever"
	PermPrazoRead        Perm = "prazos:ler"
	PermPrazoWrite       Perm = "prazos:escrever"  // registrar/editar (o prazo fica "a conferir")
	PermPrazoConfirm     Perm = "prazos:confirmar" // confirmar, cumprir, cancelar, excluir: SÓ advogado
)

var leitura = []Perm{PermHome, PermPessoasRead, PermProcRead, PermTarefaRead, PermAtendimentoRead, PermAgendaRead, PermDocRead, PermPrazoRead}

// matrix: negar por padrão. Administrador gere usuários e vê auditoria (não mexe em processos);
// advogado lê e escreve nos domínios; assistente lê tudo e escreve só em tarefas e agenda.
var matrix = func() map[Role]map[Perm]bool {
	m := map[Role]map[Perm]bool{Administrador: {}, Advogado: {}, Assistente: {}}
	for _, p := range leitura {
		m[Administrador][p], m[Advogado][p], m[Assistente][p] = true, true, true
	}
	for _, p := range []Perm{PermUsersManage, PermAuditRead, PermTarefaDelete} {
		m[Administrador][p] = true
	}
	for _, p := range []Perm{PermPessoasWrite, PermProcWrite, PermTarefaWrite, PermTarefaDelete, PermAtendimentoWrite, PermAgendaWrite, PermDocWrite, PermPrazoWrite, PermPrazoConfirm} {
		m[Advogado][p] = true
	}
	for _, p := range []Perm{PermTarefaWrite, PermAtendimentoWrite, PermAgendaWrite, PermPrazoWrite} {
		m[Assistente][p] = true
	}
	return m
}()

// Can diz se o papel tem a permissão. Papel ou permissão desconhecidos → false.
func Can(r Role, p Perm) bool { return matrix[r][p] }

// ValidRole informa se o papel existe.
func ValidRole(r Role) bool { _, ok := matrix[r]; return ok }
