build:
	go build ./main.go -o ./bin
run:
	go run ./cmd/... -b={b}
server:
	go run ./server -n=${n}
