build:
	go build -o brk brk.go
install: build
	cp brk ~/.local/bin
