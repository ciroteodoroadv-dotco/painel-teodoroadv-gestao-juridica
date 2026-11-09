$env:PAINEL_ENV = "dev"
$env:PAINEL_PUBLIC_URL = "http://localhost:8081"
$env:PAINEL_DB_HOST = "localhost"
$env:PAINEL_DB_PORT = "5432"
$env:PAINEL_DB_NAME = "painel_db"
$env:PAINEL_DB_USER = "painel_user"
$env:PAINEL_DB_PASSWORD = "devpassword"
$env:PAINEL_DB_APP_PASSWORD = "devpassword"
$env:PAINEL_SESSION_KEY = "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI="
$env:PAINEL_FIELD_KEY = "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI="
$env:PAINEL_DATAJUD_KEY = "cGFzc3dvcmQ="

Write-Host "Aplicando migracoes..."
go run ./cmd/painel migrate up

Write-Host "Iniciando servidor web na porta 8081..."
go run ./cmd/painel
