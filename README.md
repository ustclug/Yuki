ustcmirror
==========


[![Build Status](https://travis-ci.org/ustclug/ustcmirror.svg?branch=master)](https://travis-ci.org/ustclug/ustcmirror)

- [Introduction](#introduction)
- [Dependencies](#dependencies)
- [Quickstart](#quickstart)
- [API Documentation](#api-documentation)
- [Configuration](#configuration)
    - [Server side](#server-side)
    - [Client side](#client-side)

# Introduction

Aims to provide effortless management of docker containers on USTC Mirrors

# Dependencies

* Node.js > 6
* Docker
* MongoDB
* python (building dep)
* make (building dep)
* g++ (building dep)

# Quickstart

```
npm i -g ustcmirror
```

Specify the ip address to be bound in either `/etc/ustcmirror/config.json` or `~/.ustcmirror/config.json`:

```
$ cat ~/.ustcmirror/config.json
{
    "BIND_ADDRESS": "1.2.3.4",
    "LOGDIR_ROOT": "/home/knight/logs"
}
```

Start a mongo instance:

```
docker run --name mongo -p 127.0.0.1:27017:27017 -tid mongo:3.4
```

Run the daemon in debug mode:

```
NODE_ENV=dev ustcmirror daemon
```

Play with the CLI:

```
ustcmirror -h
```

# API Documentation

* [Routes](https://ustclug.github.io/ustcmirror/)

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
| `BIND_ADDRESS` | Defaults to empty. |
| `CT_LABEL` | Defaults to `syncing`. |
| `CT_NAME_PREFIX` | Defaults to `syncing`. |
| `LOGDIR_ROOT` | Defaults to `/var/log/ustcmirror`. |
| `IMAGES_UPGRADE_INTERVAL` | Defaults to `1 * * * *`. |
| `LOGLEVEL` | Defaults to `debug` if `NODE_ENV == 'dev'` else `warn`. |
| `OWNER` | Defaults to `${process.getuid()}:${process.getgid()}` |
| `TIMESTAMP` | Defaults to `true` |

### Client side

| Parameter | Description |
|-----------|-------------|
| `API_ROOT` | Defaults to `http://localhost:${API_PORT}/`. |
