SHELL=/bin/sh

EXAMPLES=$(wildcard examples/*.dat)
LDFLAGS="-s -w"
PKG=./cmd/senddat

senddat: cmd/senddat/main.go
	go build -ldflags=$(LDFLAGS) -o $@  $(PKG)

test_examples: clean senddat $(EXAMPLES)
	true $(foreach f,$(EXAMPLES),&& (printf "*** $f ***\n" && ./senddat -t $f | ./senddat -r >/dev/null))

clean:
	-rm senddat
	
