.PHONY: build build-windows clean

build:
	go build -o tidy-exif ./cmd/tidy-exif

build-windows:
	GOOS=windows GOARCH=amd64 go build -o tidy-exif.exe ./cmd/tidy-exif

clean:
	rm -f tidy-exif tidy-exif.exe
