# nyat

nyat is an email client for your terminal.

Join the IRC channel for support or to send patches, etc: [#iwakura on irc.freenode.net](http://webchat.freenode.net/?channels=iwakura&uio=d4)

## Usage

On its first run, nyat will copy the default config files to `~/.config/nyat`
on Linux or `~/Library/Preferences/nyat` on MacOS (or `$XDG_CONFIG_HOME/nyat` if set)
and show the account configuration wizard.

If you redirect stdout to a file, logging output will be written to that file:

    $ nyat > log

For instructions and documentation: see `man nyat` and further specific man
pages on there.

Note that the example HTML filter (off by default), additionally needs `w3m` and
`dante` to be installed.

## Installation

### Binary Packages

Recent versions of nyat are available on:
- [The official Gitea page of course](https://gitea.com/iwakuramarie/nyat/releases/)

Other platforms soon.

### From Source

Install the dependencies:

- go (>=1.13)
- [navidoc](https://gitea.com/iwakuramarie/navidoc)

Then compile nyat:

    $ make

nyat optionally supports notmuch. To enable it, you need to have a recent
version of [notmuch](https://notmuchmail.org/#index7h2), including the header
files (notmuch.h). Then compile nyat with the necessary build tags:

    $ GOFLAGS=-tags=notmuch make

To install nyat locally:

    # make install