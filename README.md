ustcmirror
==========


[![Build Status](https://travis-ci.org/ustclug/ustcmirror.svg?branch=master)](https://travis-ci.org/ustclug/ustcmirror)

- [Introduction](#introduction)
- [Dependencies](#dependencies)
- [How it works](#how-it-works)
- [Installation](#installation)
- [Configuration](#configuration)
    - [Server side](#server-side)
    - [Client side](#client-side)

# Introduction

Aims to provide effortless management of docker containers on USTC Mirrors

# Dependencies

* Node.js > 6
* Docker
* MongoDB

# How it works

```
docker run -i --rm --user "$OWNER" --net=host \
       -v "$storageDir:/data" -v "$LOGDIR_ROOT/$repo_name:/log" \
       -e BIND_ADDRESS=$BIND_ADDRESS [-e other env vars] \
       --label "$CT_LABEL" --name "${CT_NAME_PREFIX}-${repo_name}" \
       ustcmirror/<sync_method>:latest
```

# Installation

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
| `BIND_ADDRESS` | Defaults to empty. |
| `CT_LABEL` | Defaults to `syncing`. |
| `CT_NAME_PREFIX` | Defaults to `syncing`. |
| `LOGDIR_ROOT` | Defaults to `/var/log/ustcmirror`. |
| `IMAGES_UPGRADE_INTERVAL` | Defaults to `1 * * * *`. |
| `OWNER` | Defaults to `${process.getuid()}:${process.getgid()}` |

### Client side

| Parameter | Description |
|-----------|-------------|
| `API_ROOT` | Defaults to `http://localhost:${API_PORT}/`. |
