# holen - Application fetcher

Holen is a utility for fetching applications when you need them, on any system. For instance, if you'd like to use [gof3r](https://github.com/rlmcpherson/s3gof3r) or [hub](https://github.com/github/hub) or [terraform](https://www.terraform.io/), you would need to go to the right download page and select the right binary for your system, download it, and put it on your path (don't forget the `chmod +x`!).  Not so hard for one, but what if you need several?

Holen can help.  Running any binary that Holen knows about is as easy as:

```
holen run [app name] -- [app options]
```

And, if you'd like it to stick around for a bit, you can run this, which will put a little alias in your `$HOME/bin` directory:

```
holen link [app name]
```

Then you can just run it like it's installed.  When activated, Holen will download the right Docker image (or static binary) and then run the desired command.

Think of it as a distant cousin to Homebrew. Holen is German for "fetch".

## Manifests

Holen knows about applications via a set of manifests, which are managed in git repositories.  The main manifest repository is [here](https://github.com/holen-app/manifests), but more can be set up and managed easily.

Here is an (abbreviated) example manifest for [dockviz](https://github.com/justone/dockviz).  It teaches holen where to find the dockviz application in both the Docker and the static binary strategy:

```
desc: Visualizing Docker data
strategies:
    docker:
        image: nate/dockviz:v{{.Version}}
        docker_conn: true
        versions:
          - version: '0.5.0'
          - version: '0.4.2'
    binary:
        base_url: https://github.com/justone/dockviz/releases/download/v{{.Version}}/dockviz_{{.OSArch}}
        versions:
          - version: '0.5.0'
            os_arch:
                linux_amd64:
                    sha256sum: 0d63259921dae52329f61d63e7fef8211362d6b62396d2db5759e46054995a98
                darwin_amd64:
                    sha256sum: 3e1f72cf94ad228bd84d8d288531e28f7935346233c409c7c0cd9b88deebdc81
          - version: '0.4.2'
            os_arch:
                linux_amd64:
                    sha256sum: a8155fd4b2ebd38444c57a0e7af2892a246abec89c9f13b7421721df931b7c3b
                darwin_amd64:
                    sha256sum: 24b56b817101bc8089be3a46501ae0e5fa0a3a52fa90d640de115295427d49cf
```

## Strategies

Holen utilizes a few different strategies for fetching applications:

1. Docker image
2. Static binary
3. [cmd.io](https://cmd.io/) (experimental)

Holen will try each strategy in the above order until it is able to run the application. If you'd like it to try binary first, just run `holen config strategy.priority binary,docker`.

# Quick Start

1. [Download the latest release](https://github.com/justone/holen/releases) for your platform and place it in your \$PATH.
2. Run `holen list` to view the available commands.
3. Run `holen link [utility] -b [dir to link into]` to link a utility.  For example, `holen link jq -b $HOME/bin`.
4. Run the utility.

For more information, go to the [documentation](http://holen.endot.org).

# Development

In a Go 1.8 development environment (may I suggest [skeg](http://skeg.io/)?):

```
$ go get github.com/holen-app/holen
$ cd $GOPATH/github.com/holen-app/holen
$ go build
$ go test
```

## Contributing

See the [CONTRIBUTING.md](CONTRIBUTING.md) file for details.

# Built With

## Libraries

* [go-flags](https://github.com/jessevdk/go-flags)
* [errors](https://github.com/pkg/errors)
* [logrus](https://github.com/Sirupsen/logrus)
* [archiver](https://github.com/mholt/archiver)
* [osext](https://github.com/kardianos/osext)
* [pretty](https://github.com/kr/pretty)
* [go-homedir](https://github.com/mitchellh/go-homedir)
* [assert](https://github.com/stretchr/testify/assert)

## Tools

* [govendor](https://github.com/kardianos/govendor)
* [github-release](https://github.com/aktau/github-release)
* [gox](https://github.com/mitchellh/gox)

# License

Copyright 2016-2017 Nate Jones Licensed under the Apache License, Version 2.0.
