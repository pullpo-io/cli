CGO_CPPFLAGS ?= ${CPPFLAGS}
export CGO_CPPFLAGS
CGO_CFLAGS ?= ${CFLAGS}
export CGO_CFLAGS
CGO_LDFLAGS ?= $(filter -g -L% -l% -O%,${LDFLAGS})
export CGO_LDFLAGS

EXE =
ifeq ($(shell go env GOOS),windows)
EXE = .exe
endif

## The following tasks delegate to `script/build.go` so they can be run cross-platform.

.PHONY: bin/pullpo$(EXE)
bin/pullpo$(EXE): script/build$(EXE)
	@script/build$(EXE) $@

script/build$(EXE): script/build.go
ifeq ($(EXE),)
	GOOS= GOARCH= GOARM= GOFLAGS= CGO_ENABLED= go build -o $@ $<
else
	go build -o $@ $<
endif

.PHONY: clean
clean: script/build$(EXE)
	@$< $@

.PHONY: manpages
manpages: script/build$(EXE)
	@$< $@

.PHONY: completions
completions: bin/pullpo$(EXE)
	mkdir -p ./share/bash-completion/completions ./share/fish/vendor_completions.d ./share/zsh/site-functions
	bin/pullpo$(EXE) completion -s bash > ./share/bash-completion/completions/gh
	bin/pullpo$(EXE) completion -s fish > ./share/fish/vendor_completions.d/gh.fish
	bin/pullpo$(EXE) completion -s zsh > ./share/zsh/site-functions/_gh

# just a convenience task around `go test`
.PHONY: test
test:
	go test ./...

## Site-related tasks are exclusively intended for use by the GitHub CLI team and for our release automation.

site:
	git clone https://github.com/github/cli.github.com.git "$@"

.PHONY: site-docs
site-docs: site
	git -C site pull
	git -C site rm 'manual/gh*.md' 2>/dev/null || true
	go run ./cmd/gen-docs --website --doc-path site/manual
	rm -f site/manual/*.bak
	git -C site add 'manual/gh*.md'
	git -C site commit -m 'update help docs' || true

.PHONY: site-bump
site-bump: site-docs
ifndef GITHUB_REF
	$(error GITHUB_REF is not set)
endif
	sed -i.bak -E 's/(assign version = )".+"/\1"$(GITHUB_REF:refs/tags/v%=%)"/' site/index.html
	rm -f site/index.html.bak
	git -C site commit -m '$(GITHUB_REF:refs/tags/v%=%)' index.html

## Install/uninstall tasks are here for use on *nix platform. On Windows, there is no equivalent.

DESTDIR :=
prefix  := /usr/local
bindir  := ${prefix}/bin
datadir := ${prefix}/share
mandir  := ${datadir}/man

.PHONY: install
install: bin/pullpo manpages completions
	install -d ${DESTDIR}${bindir}
	install -m755 bin/pullpo ${DESTDIR}${bindir}/
	install -d ${DESTDIR}${mandir}/man1
	install -m644 ./share/man/man1/* ${DESTDIR}${mandir}/man1/
	install -d ${DESTDIR}${datadir}/bash-completion/completions
	install -m644 ./share/bash-completion/completions/pullpo ${DESTDIR}${datadir}/bash-completion/completions/gh
	install -d ${DESTDIR}${datadir}/fish/vendor_completions.d
	install -m644 ./share/fish/vendor_completions.d/gh.fish ${DESTDIR}${datadir}/fish/vendor_completions.d/gh.fish
	install -d ${DESTDIR}${datadir}/zsh/site-functions
	install -m644 ./share/zsh/site-functions/_pullpo ${DESTDIR}${datadir}/zsh/site-functions/_gh

.PHONY: uninstall
uninstall:
	rm -f ${DESTDIR}${bindir}/pullpo ${DESTDIR}${mandir}/man1/gh.1 ${DESTDIR}${mandir}/man1/gh-*.1
	rm -f ${DESTDIR}${datadir}/bash-completion/completions/gh
	rm -f ${DESTDIR}${datadir}/fish/vendor_completions.d/gh.fish
	rm -f ${DESTDIR}${datadir}/zsh/site-functions/_gh
