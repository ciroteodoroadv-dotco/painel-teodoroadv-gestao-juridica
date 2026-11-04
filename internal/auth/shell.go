package auth

import (
	"context"
	"sort"
	"strings"
	"time"
)

// Head descreve o cabeçalho da página no padrão do painel B (.topbar).
type Head struct {
	Title, Sub   string // h1 e <p class="sub">
	Chip         string // data ISO de hoje (só a Agenda): vira <time class="agenda-date-chip">
	ActionLabel  string // "Novo compromisso", "Nova tarefa"… vazio = sem botão
	ActionHref   string // caminho da página (ou do fragmento do drawer)
	ActionDrawer bool   // true: o botão ganha data-drawer="<ActionHref>"
}

var diasSemana = [...]string{"dom.", "seg.", "ter.", "qua.", "qui.", "sex.", "sáb."}

// ChipLabel formata a data como o Intl.DateTimeFormat(pt-BR) do B: "ter., 03/11/2026".
func (h Head) ChipLabel() string {
	t, err := time.Parse("2006-01-02", h.Chip)
	if err != nil {
		return h.Chip
	}
	return diasSemana[t.Weekday()] + ", " + t.Format("02/01/2006")
}

func hojeISO(now time.Time) string {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.FixedZone("BRT", -3*3600)
	}
	return now.In(loc).Format("2006-01-02")
}

// NavLink é um item do menu lateral já filtrado pelo papel.
type NavLink struct {
	Label, Href string
	Count       int
	HasCount    bool
	Active      bool
	Drawer      bool // item de ação: abre o fragmento Href num drawer (sem página própria)
}

type navItem struct {
	label, href string
	perm        Perm
	count       func(context.Context) (int, error)
	sistema     bool
	drawer      bool
}

// Faixa é a faixa de aviso (.notice) no topo da Agenda inicial.
type Faixa struct{ Strong, Text, Href, Link string }

// FaixaFunc produz a faixa de um domínio; faixa vazia (Strong e Text vazios) não é exibida.
type FaixaFunc func(ctx context.Context, s *Session) (Faixa, error)

// AddNav registra um item do menu principal, sem contador.
func (a *Auth) AddNav(label, href string, perm Perm) { a.AddNavCount(label, href, perm, nil) }

// AddNavCount registra um item do menu principal com contador (o <span class="count"> do B).
func (a *Auth) AddNavCount(label, href string, perm Perm, count func(context.Context) (int, error)) {
	a.nav = append(a.nav, navItem{label: label, href: href, perm: perm, count: count})
}

// AddNavAcao registra um item do menu principal que abre um drawer (ex.: "Calculadora de prazo"), sem contador.
func (a *Auth) AddNavAcao(label, href string, perm Perm) {
	a.nav = append(a.nav, navItem{label: label, href: href, perm: perm, drawer: true})
}

// AddNavSistema registra um item do segundo grupo do menu (o que o v2 tem e o B não).
func (a *Auth) AddNavSistema(label, href string, perm Perm) {
	a.nav = append(a.nav, navItem{label: label, href: href, perm: perm, sistema: true})
}

// AddFaixa registra uma faixa de aviso para a Agenda inicial.
func (a *Auth) AddFaixa(f FaixaFunc) { a.faixas = append(a.faixas, f) }

// tabOf devolve a aba do B (agenda, pessoas…) que o caminho representa; a raiz é a Agenda.
func tabOf(path string) string {
	if path == "/" {
		return "agenda"
	}
	return strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)[0]
}

// menu monta os dois grupos do menu: principal em ordem alfabética (como no B) e sistema.
func (a *Auth) menu(ctx context.Context, role Role, path string) (principal, sistema []NavLink) {
	for _, n := range a.nav {
		if !Can(role, n.perm) {
			continue
		}
		l := NavLink{Label: n.label, Href: n.href, Active: !n.drawer && ativo(path, n.href), Drawer: n.drawer}
		if n.count != nil && ctx != nil {
			if c, err := n.count(ctx); err == nil {
				l.Count, l.HasCount = c, true
			}
		}
		if n.sistema {
			sistema = append(sistema, l)
		} else {
			principal = append(principal, l)
		}
	}
	sort.SliceStable(principal, func(i, j int) bool { return principal[i].Label < principal[j].Label })
	return principal, sistema
}

func ativo(path, href string) bool {
	return path == href || strings.HasPrefix(path, href+"/") || (path == "/" && href == "/agenda")
}

// sistemaFixos acrescenta ao segundo grupo os itens próprios da conta e da administração.
func sistemaFixos(sistema []NavLink, role Role, path string) []NavLink {
	sistema = append(sistema, NavLink{Label: "Minha conta", Href: "/conta", Active: ativo(path, "/conta")})
	if Can(role, PermUsersManage) {
		sistema = append(sistema, NavLink{Label: "Usuários", Href: "/admin/usuarios", Active: ativo(path, "/admin/usuarios")})
	}
	if Can(role, PermAuditRead) {
		sistema = append(sistema, NavLink{Label: "Auditoria", Href: "/admin/auditoria", Active: ativo(path, "/admin/auditoria")})
	}
	return sistema
}
