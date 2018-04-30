README
=======

[![Build Status](https://travis-ci.org/knight42/Yuki.svg?branch=master)](https://travis-ci.org/knight42/Yuki)
[![Go Report](https://goreportcard.com/badge/github.com/knight42/Yuki)](https://goreportcard.com/report/github.com/knight42/Yuki)

- [Requirements](#requirements)
- [Quickstart](#quickstart)
- [CLI](#cli)

Sync local repositories with remote.

## Requirements

* Docker
* MongoDB

## Quickstart

Download the binary from the [Release](https://github.com/knight42/Yuki/releases) page:

```
For example:
$ wget https://github.com/knight42/Yuki/releases/download/v0.1.0/yukid-v0.1.0-linux-amd64.tar.gz
```

Configure yukid:

```
# mkdir /etc/yuki/
# chown mirror:mirror /etc/yuki
$ curl 'https://raw.githubusercontent.com/knight42/Yuki/master/dist/daemon.toml' > /etc/yuki/daemon.toml
$ vim /etc/yuki/daemon.toml
```

Run MongoDB:

```
$ docker run --name mongo -tid mongo:3.6
```

Create systemd service:
```
# curl 'https://raw.githubusercontent.com/knight42/Yuki/master/dist/yukid.service' > /etc/systemd/system/yukid.service
```

Start yukid:
```
# systemctl enable yukid
# systemctl start yukid
```

## CLI

[Yuki-cli](https://github.com/knight42/Yuki-cli)
