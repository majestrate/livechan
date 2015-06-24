all:
	go build -o livechan.bin src/*.go
clean:
	rm -f livechan.bin
