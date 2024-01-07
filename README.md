README
=======

[![Presubmit Checks](https://github.com/ustclug/Yuki/actions/workflows/pr-presubmit-checks.yml/badge.svg)](https://github.com/ustclug/Yuki/actions/workflows/pr-presubmit-checks.yml)
[![Go Report](https://goreportcard.com/badge/github.com/ustclug/Yuki)](https://goreportcard.com/report/github.com/ustclug/Yuki)

- [Requirements](#requirements)
- [Quickstart](#quickstart)
- [Handbook](#handbook)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

Sync local repositories with remote.

## Requirements

* Docker
* SQLite

## Quickstart

Download the binaries from the [Release](https://github.com/ustclug/Yuki/releases) page. For example:

```bash
wget https://github.com/ustclug/Yuki/releases/latest/download/yukictl_linux_amd64
wget https://github.com/ustclug/Yuki/releases/latest/download/yukid_linux_amd64

sudo cp yukictl_linux_amd64 /usr/local/bin/yukictl
sudo cp yukid_linux_amd64 /usr/local/bin/yukid
```

Configure yukid:

```bash
sudo mkdir /etc/yuki/
sudo useradd -m mirror
mkdir /tmp/repo-logs/ /tmp/repo-configs/

cat <<EOF | sudo tee /etc/yuki/daemon.toml
db_url = "/tmp/yukid.db"
# uid:gid
owner = "$(id -u mirror):$(id -g mirror)"
repo_logs_dir = "/tmp/repo-logs/"
repo_config_dir = "/tmp/repo-configs/"
EOF
```

Configure systemd service:

```bash
curl 'https://raw.githubusercontent.com/ustclug/Yuki/main/deploy/yukid.service' | sudo tee /etc/systemd/system/yukid.service
systemctl enable yukid
systemctl start yukid
systemctl status yukid
```

Setup repository:

```bash
# The repository directory must be created in advance
mkdir /tmp/repo-data/docker-ce

# Sync docker-ce repository from rsync.mirrors.ustc.edu.cn
cat <<EOF > /tmp/repo-configs/docker-ce.yaml
name: docker-ce
# every 1 hour
cron: "0 * * * *"
storageDir: /tmp/repo-data/docker-ce
image: ustcmirror/rsync:latest
logRotCycle: 2
envs:
  RSYNC_HOST: rsync.mirrors.ustc.edu.cn
  RSYNC_PATH: docker-ce/
  RSYNC_EXCLUDE: --exclude=.~tmp~/
  RSYNC_EXTRA: --size-only
  RSYNC_MAXDELETE: "50000"
EOF

yukictl reload
# Verify
yukictl repo ls

# Trigger synchronization immediately
yukictl sync docker-ce
```

For more details of the configuration file, please refer to the [yukid handbook](./cmd/yukid/README.md).

## Handbook

* [yukid](./cmd/yukid/README.md): Yuki daemon
* [yukictl](./cmd/yukictl/README.md): Yuki cli

## Migration Guide

### v0.3.x -> v0.4.x

For configuration:

```bash
sed -i.bak 's/log_dir/repo_logs_dir/' /etc/yuki/daemon.toml
# Also remember to update the `images_upgrade_interval` field in /etc/yuki/daemon.toml if it is set.

sed -i.bak 's/interval/cron/' /path/to/repo/configs/*.yaml
```

For post sync hook, the environment variables that are passed to the hook script are changed:
* `Dir` -> `DIR`: the directory of the repository
* `Name` -> `NAME`: the name of the repository

## Troubleshooting

### version `GLIBC_2.XX' not found

You might encounter the following error when running yukid:

```
$ ./yukid -V
./yukid: /lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.33' not found (required by ./yukid)
./yukid: /lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.32' not found (required by ./yukid)
./yukid: /lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.34' not found (required by ./yukid)
```

This is because `yukid` is complied with CGO enabled, which is required by https://github.com/mattn/go-sqlite3.
The version of glibc that is linked to `yukid` might differ from the actual one that exists on your current machine.
You will need to compile `yukid` on your current machine or run `yukid` in container.

Tips:
* To check your current glibc version:
```
$ /lib/x86_64-linux-gnu/libc.so.6 | grep -i glibc
```
* The docker images of `yukid`: https://github.com/ustclug/Yuki/pkgs/container/yukid

## Development

* Build `yukid`:

```
make yukid
```

* Build `yukictl`:

```
make yukictl
```

* Lint the whole project:

```
make lint
```
