set PKGNAME=github.com/oneumyvakin/rpmeta
set LOCALPATH=%~dp0

goimports.exe -w .
go fmt %PKGNAME%
staticcheck.exe %PKGNAME%
go vet %PKGNAME%

set GOOS=linux
set GOARCH=amd64
go build -o rpmeta.%GOARCH% %PKGNAME%

set GOOS=windows
set GOARCH=amd64
go build -o rpmeta.exe %PKGNAME%