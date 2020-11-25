# Available architecture combinations for Go: https://golang.org/doc/install/source#environment

VERSION ?= dev

linux-amd64:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -o out_newrelic-linux-amd64-${VERSION}.so .

windows-amd64:
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ go build -buildmode=c-shared -o out_newrelic-windows-amd64.dll .

windows-386:
	CGO_ENABLED=1 GOOS=windows GOARCH=386 CC=i686-w64-mingw32-gcc CXX=i686-w64-mingw32-g++ go build -buildmode=c-shared -o out_newrelic-windows-386.dll .

linux-arm64:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ go build -buildmode=c-shared -o out_newrelic-linux-arm64.so .

linux-arm:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm CC=arm-linux-gnueabihf-gcc CXX=arm-linux-gnueabihf-g++ go build -buildmode=c-shared -o out_newrelic-linux-arm.so .

clean:
	rm -rf *.so *.h *~
