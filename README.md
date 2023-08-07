README
=======

[![Build Status](https://github.com/ustclug/Yuki/workflows/build/badge.svg)](https://github.com/ustclug/Yuki/actions)
[![Go Report](https://goreportcard.com/badge/github.com/ustclug/Yuki)](https://goreportcard.com/report/github.com/ustclug/Yuki)

- [Requirements](#requirements)
- [Quickstart](#quickstart)
- [Handbook](#handbook)

Sync local repositories with remote.

## Requirements

* Docker
* MongoDB

## Quickstart

Download the binary from the [Release](https://github.com/ustclug/Yuki/releases) page, for example:

```
$ wget https://github.com/ustclug/Yuki/releases/download/v0.2.2/yuki-v0.2.2-linux-amd64.tar.gz
```

Configure yukid:

```
# mkdir /etc/yuki/
# chown mirror:mirror /etc/yuki
$ curl 'https://raw.githubusercontent.com/ustclug/Yuki/master/dist/daemon.toml' > /etc/yuki/daemon.toml
$ vim /etc/yuki/daemon.toml
```

Run MongoDB:

```
$ docker run -p 127.0.0.1:27017:27017 -tid --name mongo mongo:3.6
```

Create systemd service:
```
# curl 'https://raw.githubusercontent.com/ustclug/Yuki/master/dist/yukid.service' > /etc/systemd/system/yukid.service
```

Start yukid:
```
# systemctl enable yukid
# systemctl start yukid
```

## Handbook

* [yukid](./cmd/yukid/README.md): Yuki daemon
* [yukictl](./cmd/yukictl/README.md): Yuki cli
