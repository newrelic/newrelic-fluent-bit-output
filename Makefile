all:
	go build -buildmode=c-shared -o out_newrelic.so .
	go build -buildmode=c-shared -o out_newrelic.dll .

fast:
	go build out_newrelic.go

clean:
	rm -rf *.so *.h *~
