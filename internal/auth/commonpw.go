package auth

// commonPasswords é uma lista MÍNIMA de senhas notoriamente vazadas com 12+ caracteres.
// Limite conhecido: substituir por lista maior (ex.: top 100 mil) antes de produção — ver plano M2.
var commonPasswords = func() map[string]bool {
	m := map[string]bool{}
	for _, p := range []string{
		"123456789012", "1234567890123", "123456789abc", "qwertyuiopas", "qwertyuiop12",
		"password1234", "password12345", "passw0rd1234", "senha1234567", "senha12345678",
		"mudar1234567", "mudar@123456", "admin1234567", "administrator", "abcdefghijkl",
		"abc123456789", "iloveyou1234", "letmein12345", "welcome12345", "changeme1234",
		"qazwsxedcrfv", "1q2w3e4r5t6y", "1qaz2wsx3edc", "000000000000", "111111111111",
		"brasil123456", "flamengo1234", "corinthians1", "123mudar12345", "trocar123456",
	} {
		m[p] = true
	}
	return m
}()
