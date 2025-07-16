.PHONY: build clean

build:
	go build -o healthcheck .

clean:
	rm -f healthcheck
