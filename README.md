Yuki
=======


[![Build Status](https://travis-ci.org/ustclug/ustcmirror.svg?branch=master)](https://travis-ci.org/ustclug/ustcmirror)

- [Introduction](#introduction)
- [Installation](#installation)
- [Configuration](#configuration)
    - [Server side](#server-side)
    - [Client side](#client-side)

# Introduction

Aimed to provide effortless management of docker containers on USTC Mirrors

# Installation

```
git clone https://github.com/ustclug/ustcmirror && cd ustcmirror
npm i
npm run build
npm link
```

Specify the ip address to be bound in either `/etc/ustcmirror/config.json` or `~/.ustcmirror/config.json`:

```
$ cat ~/.ustcmirror/config.json
{
    "BIND_ADDR": "1.2.3.4",
    "LOGDIR_ROOT": "/home/knight/logs"
}
```

Run the daemon in debug mode:

```
npm run yukid:dev
```

Play with the CLI:

```
ustcmirror -h
```

# Configuration

Global configuration: `/etc/ustcmirror/config.(js|json)`

User-specific configuration: `~/.ustcmirror/config.(js|json)`

### Server side

| Parameter | Description |
|-----------|-------------|
| `DB_USER` | Defaults to empty. |
| `DB_PASSWD` | Defaults to empty. |
| `DB_HOST` | Defaults to `127.0.0.1`. |
| `DB_NAME` | Defaults to `mirror`. |
| `DB_PORT` | Defaults to `27017`. |
| `API_PORT` | Defaults to `9999`. |
| `DOCKERD_PORT` | Defaults to `2375`. |
| `DOCKERD_HOST` | Defaults to `127.0.0.1`. |
| `DOCKERD_SOCKET` | Defaults to `/var/run/docker.sock`. |
| `BIND_ADDR` | Defaults to empty. |
| `CT_LABEL` | Defaults to `syncing`. |
| `CT_NAME_PREFIX` | Defaults to `syncing`. |
| `LOGDIR_ROOT` | Defaults to `/var/log/ustcmirror`. |
| `OWNER` | Defaults to `${process.getuid()}:${process.getgid()}` |

### Client side

| Parameter | Description |
|-----------|-------------|
| `API_ROOT` | Defaults to `http://localhost:${API_PORT}`. |
