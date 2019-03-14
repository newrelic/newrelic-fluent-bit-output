all:
	go build -buildmode=c-shared -o out_newrelic.so .

fast:
	go build out_newrelic.go

clean:
	rm -rf *.so *.h *~
