echo %cd%
cd /d ./
set GOPATH=%cd%
svn update src
del chat.exe
go build -o apps

pause