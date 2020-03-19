all:
	go build -buildmode=c-shared -o out_newrelic.so .

win64:
	go build -buildmode=c-shared -o out_newrelic_win64.dll .

win32:
	go build -buildmode=c-shared -o out_newrelic_win32.dll .

fast:
	go build out_newrelic.go

clean:
	rm -rf *.so *.h *~
