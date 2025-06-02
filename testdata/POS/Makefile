MAIN=../../cmd/senddat

ALL_DAT=$(wildcard *.dat)
ALL_PRN=$(ALL_DAT:.dat=.prn)
$(info $(ALL_PRN))

all: $(ALL_PRN)
.PHONY: all

%.prn: %.dat
	go run $(MAIN) -o $@ $^

clean:
	-rm $(ALL_PRN)