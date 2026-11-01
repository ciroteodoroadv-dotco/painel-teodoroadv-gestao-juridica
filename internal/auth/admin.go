package auth

import (
	"net/http"
	"strings"

	"painelteodoro/internal/audit"
)

var avisosAdmin = map[string]string{
	"criado":   "Usuário criado com senha provisória: a troca será exigida no primeiro acesso.",
	"alterado": "Alteração registrada na auditoria.",
}

// RegisterAdmin liga a gestão de usuários (somente administrador).
func (a *Auth) RegisterAdmin(rt *Router) {
	rt.Handle("GET /admin/usuarios", PermUsersManage, a.usuariosGet)
	rt.Handle("POST /admin/usuarios", PermUsersManage, a.usuarioCriar)
	rt.Handle("POST /admin/usuarios/{id}/papel", PermUsersManage, a.usuarioPapel)
	rt.Handle("POST /admin/usuarios/{id}/ativo", PermUsersManage, a.usuarioAtivo)
	rt.Handle("POST /admin/usuarios/{id}/desbloquear", PermUsersManage, a.usuarioDesbloquear)
	rt.Handle("POST /admin/usuarios/{id}/senha", PermUsersManage, a.usuarioSenha)
}

func (a *Auth) usuariosPage(w http.ResponseWriter, r *http.Request, s *Session, status int, notice, errMsg string) {
	users, err := a.Store.ListUsers(r.Context(), a.now())
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	a.render(w, r, status, "usuarios.html", PageData{Title: "Usuários", Email: s.Email, Role: string(s.Role),
		Users: users, Notice: notice, Error: errMsg})
}

func (a *Auth) usuariosGet(w http.ResponseWriter, r *http.Request, s *Session) {
	a.usuariosPage(w, r, s, http.StatusOK, avisosAdmin[r.URL.Query().Get("aviso")], "")
}

// provisoria valida e gera o hash de uma senha provisória.
func provisoria(pw, email string) (string, string) {
	if err := CheckPolicy(pw, email); err != nil {
		return "", capitalize(err.Error()) + "."
	}
	h, err := HashPassword(pw)
	if err != nil {
		return "", "Erro interno."
	}
	return h, ""
}

func (a *Auth) usuarioCriar(w http.ResponseWriter, r *http.Request, s *Session) {
	ctx := r.Context()
	email := strings.TrimSpace(r.PostFormValue("email"))
	role := Role(r.PostFormValue("papel"))
	if !strings.Contains(email, "@") || !ValidRole(role) {
		a.usuariosPage(w, r, s, http.StatusBadRequest, "", "Informe um e-mail válido e um papel.")
		return
	}
	hash, msg := provisoria(r.PostFormValue("senha"), email)
	if msg != "" {
		a.usuariosPage(w, r, s, http.StatusBadRequest, "", msg)
		return
	}
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id, err := a.Store.CreateUser(ctx, tx, email, role, hash, true)
	if err != nil {
		a.usuariosPage(w, r, s, http.StatusConflict, "", "Não foi possível criar (esse e-mail já existe?).")
		return
	}
	if err = a.audit(r, w, tx, audit.Event{Actor: s.UserID, Action: "usuario.criado", TargetType: "usuario",
		TargetID: id, Result: audit.OK, Meta: map[string]string{"papel": string(role)}}); err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/usuarios?aviso=criado", http.StatusSeeOther)
}

// mexer aplica uma alteração em OUTRO usuário, auditada, e volta para a lista.
func (a *Auth) mexer(w http.ResponseWriter, r *http.Request, s *Session, action string, meta map[string]string,
	do func(tx txLike, id string) error) {
	ctx := r.Context()
	id := r.PathValue("id")
	if id == s.UserID {
		a.usuariosPage(w, r, s, http.StatusConflict, "", "Por segurança, você não altera a si mesmo por aqui (use Minha conta).")
		return
	}
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = do(tx, id); err == nil {
		err = a.audit(r, w, tx, audit.Event{Actor: s.UserID, Action: action, TargetType: "usuario", TargetID: id,
			Result: audit.OK, Meta: meta})
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		http.Error(w, "Usuário não encontrado.", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, "/admin/usuarios?aviso=alterado", http.StatusSeeOther)
}

func (a *Auth) usuarioPapel(w http.ResponseWriter, r *http.Request, s *Session) {
	role := Role(r.PostFormValue("papel"))
	if !ValidRole(role) {
		a.usuariosPage(w, r, s, http.StatusBadRequest, "", "Papel inválido.")
		return
	}
	a.mexer(w, r, s, "usuario.papel_alterado", map[string]string{"papel": string(role)},
		func(tx txLike, id string) error { return a.Store.UpdateRole(r.Context(), tx, id, role) })
}

func (a *Auth) usuarioAtivo(w http.ResponseWriter, r *http.Request, s *Session) {
	on := r.PostFormValue("ativo") == "1"
	action := "usuario.desativado"
	if on {
		action = "usuario.reativado"
	}
	a.mexer(w, r, s, action, nil, func(tx txLike, id string) error { return a.Store.SetActive(r.Context(), tx, id, on) })
}

func (a *Auth) usuarioDesbloquear(w http.ResponseWriter, r *http.Request, s *Session) {
	a.mexer(w, r, s, "usuario.desbloqueado", nil, func(tx txLike, id string) error { return a.Store.Unlock(r.Context(), tx, id) })
}

func (a *Auth) usuarioSenha(w http.ResponseWriter, r *http.Request, s *Session) {
	u, err := a.Store.UserByID(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, "Usuário não encontrado.", http.StatusNotFound)
		return
	}
	hash, msg := provisoria(r.PostFormValue("senha"), u.Email)
	if msg != "" {
		a.usuariosPage(w, r, s, http.StatusBadRequest, "", msg)
		return
	}
	a.mexer(w, r, s, "usuario.senha_redefinida", nil,
		func(tx txLike, id string) error { return a.Store.SetPassword(r.Context(), tx, id, hash, true) })
}
