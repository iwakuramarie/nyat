.POSIX:
.SUFFIXES:
.SUFFIXES: .1 .5 .7 .1.nvd .5.nvd .7.nvd

VERSION=0.5.2

VPATH=doc
PREFIX?=/usr/local
BINDIR?=$(PREFIX)/bin
SHAREDIR?=$(PREFIX)/share/nyat
MANDIR?=$(PREFIX)/share/man
GO?=go
GOFLAGS?=

GOSRC:=$(shell find . -name '*.go')
GOSRC+=go.mod go.sum

nyat: $(GOSRC)
	$(GO) build $(GOFLAGS) \
		-ldflags "-X main.Prefix=$(PREFIX) \
		-X main.ShareDir=$(SHAREDIR) \
		-X main.Version=$(VERSION)" \
		-o $@

nyat.conf: config/nyat.conf.in
	sed -e 's:@SHAREDIR@:$(SHAREDIR):g' > $@ < config/nyat.conf.in

debug: $(GOSRC)
	GOFLAGS="-tags=notmuch" \
	dlv debug --headless --listen localhost:4747 &>/dev/null

DOCS := \
	nyat.1 \
	nyat-search.1 \
	nyat-config.5 \
	nyat-imap.5 \
	nyat-maildir.5 \
	nyat-sendmail.5 \
	nyat-notmuch.5 \
	nyat-smtp.5 \
	nyat-tutorial.7 \
	nyat-templates.7 \
	nyat-stylesets.7

.1.nvd.1:
	navidoc < $< > $@

.5.nvd.5:
	navidoc < $< > $@

.7.nvd.7:
	navidoc < $< > $@

doc: $(DOCS)

all: nyat nyat.conf doc

# Exists in GNUMake but not in NetBSD make and others.
RM?=rm -f

clean:
	$(RM) $(DOCS) nyat.conf nyat

install: all
	mkdir -m755 -p $(DESTDIR)$(BINDIR) $(DESTDIR)$(MANDIR)/man1 $(DESTDIR)$(MANDIR)/man5 $(DESTDIR)$(MANDIR)/man7 \
		$(DESTDIR)$(SHAREDIR) $(DESTDIR)$(SHAREDIR)/filters $(DESTDIR)$(SHAREDIR)/templates $(DESTDIR)$(SHAREDIR)/stylesets
	install -m755 nyat $(DESTDIR)$(BINDIR)/nyat
	install -m644 nyat.1 $(DESTDIR)$(MANDIR)/man1/nyat.1
	install -m644 nyat-search.1 $(DESTDIR)$(MANDIR)/man1/nyat-search.1
	install -m644 nyat-config.5 $(DESTDIR)$(MANDIR)/man5/nyat-config.5
	install -m644 nyat-imap.5 $(DESTDIR)$(MANDIR)/man5/nyat-imap.5
	install -m644 nyat-maildir.5 $(DESTDIR)$(MANDIR)/man5/nyat-maildir.5
	install -m644 nyat-sendmail.5 $(DESTDIR)$(MANDIR)/man5/nyat-sendmail.5
	install -m644 nyat-notmuch.5 $(DESTDIR)$(MANDIR)/man5/nyat-notmuch.5
	install -m644 nyat-smtp.5 $(DESTDIR)$(MANDIR)/man5/nyat-smtp.5
	install -m644 nyat-tutorial.7 $(DESTDIR)$(MANDIR)/man7/nyat-tutorial.7
	install -m644 nyat-templates.7 $(DESTDIR)$(MANDIR)/man7/nyat-templates.7
	install -m644 nyat-stylesets.7 $(DESTDIR)$(MANDIR)/man7/nyat-stylesets.7
	install -m644 config/accounts.conf $(DESTDIR)$(SHAREDIR)/accounts.conf
	install -m644 nyat.conf $(DESTDIR)$(SHAREDIR)/nyat.conf
	install -m644 config/binds.conf $(DESTDIR)$(SHAREDIR)/binds.conf
	install -m755 filters/hldiff $(DESTDIR)$(SHAREDIR)/filters/hldiff
	install -m755 filters/html $(DESTDIR)$(SHAREDIR)/filters/html
	install -m755 filters/plaintext $(DESTDIR)$(SHAREDIR)/filters/plaintext
	install -m644 templates/quoted_reply $(DESTDIR)$(SHAREDIR)/templates/quoted_reply
	install -m644 templates/forward_as_body $(DESTDIR)$(SHAREDIR)/templates/forward_as_body
	install -m644 config/default_styleset $(DESTDIR)$(SHAREDIR)/stylesets/default

RMDIR_IF_EMPTY:=sh -c '\
if test -d $$0 && ! ls -1qA $$0 | grep -q . ; then \
	rmdir $$0; \
fi'

uninstall:
	$(RM) $(DESTDIR)$(BINDIR)/nyat
	$(RM) $(DESTDIR)$(MANDIR)/man1/nyat.1
	$(RM) $(DESTDIR)$(MANDIR)/man1/nyat-search.1
	$(RM) $(DESTDIR)$(MANDIR)/man5/nyat-config.5
	$(RM) $(DESTDIR)$(MANDIR)/man5/nyat-imap.5
	$(RM) $(DESTDIR)$(MANDIR)/man5/nyat-maildir.5
	$(RM) $(DESTDIR)$(MANDIR)/man5/nyat-sendmail.5
	$(RM) $(DESTDIR)$(MANDIR)/man5/nyat-notmuch.5
	$(RM) $(DESTDIR)$(MANDIR)/man5/nyat-smtp.5
	$(RM) $(DESTDIR)$(MANDIR)/man7/nyat-tutorial.7
	$(RM) $(DESTDIR)$(MANDIR)/man7/nyat-templates.7
	$(RM) $(DESTDIR)$(MANDIR)/man7/nyat-stylesets.7
	$(RM) -r $(DESTDIR)$(SHAREDIR)
	${RMDIR_IF_EMPTY} $(DESTDIR)$(BINDIR)
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man1
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man5
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man7
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)

.DEFAULT_GOAL := all

.PHONY: all doc clean install uninstall debug
