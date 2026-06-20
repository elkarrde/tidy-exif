.PHONY: build build-windows build-arm64 dist clean

VERSION := $(shell sed -n 's/^[[:space:]]*version[[:space:]]*=[[:space:]]*"\(.*\)".*/\1/p' cmd/tidy-exif/main.go)
DISTDIR := dist

build:
	go build -o tidy-exif ./cmd/tidy-exif

build-windows:
	GOOS=windows GOARCH=amd64 go build -o tidy-exif.exe ./cmd/tidy-exif

build-arm64:
	GOOS=linux GOARCH=arm64 go build -o tidy-exif-arm64 ./cmd/tidy-exif

# Package binaries with LICENSE and README for distribution.
# Linux: tar.gz, Windows: zip. Named with the version from main.go.
dist: build build-windows build-arm64
	rm -rf $(DISTDIR)
	mkdir -p $(DISTDIR)/tidy-exif-$(VERSION)-linux-amd64
	cp tidy-exif LICENSE README.md $(DISTDIR)/tidy-exif-$(VERSION)-linux-amd64/
	tar -C $(DISTDIR) -czf $(DISTDIR)/tidy-exif-$(VERSION)-linux-amd64.tar.gz tidy-exif-$(VERSION)-linux-amd64
	mkdir -p $(DISTDIR)/tidy-exif-$(VERSION)-linux-arm64
	cp tidy-exif-arm64 $(DISTDIR)/tidy-exif-$(VERSION)-linux-arm64/tidy-exif
	cp LICENSE README.md $(DISTDIR)/tidy-exif-$(VERSION)-linux-arm64/
	tar -C $(DISTDIR) -czf $(DISTDIR)/tidy-exif-$(VERSION)-linux-arm64.tar.gz tidy-exif-$(VERSION)-linux-arm64
	mkdir -p $(DISTDIR)/tidy-exif-$(VERSION)-windows-amd64
	cp tidy-exif.exe LICENSE README.md $(DISTDIR)/tidy-exif-$(VERSION)-windows-amd64/
	cd $(DISTDIR) && zip -qr tidy-exif-$(VERSION)-windows-amd64.zip tidy-exif-$(VERSION)-windows-amd64
	rm -rf $(DISTDIR)/tidy-exif-$(VERSION)-linux-amd64 $(DISTDIR)/tidy-exif-$(VERSION)-linux-arm64 $(DISTDIR)/tidy-exif-$(VERSION)-windows-amd64
	@echo "Created:"
	@ls -1 $(DISTDIR)/*.tar.gz $(DISTDIR)/*.zip

clean:
	rm -f tidy-exif tidy-exif.exe tidy-exif-arm64
	rm -rf $(DISTDIR)
