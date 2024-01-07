README
=======

[![Presubmit Checks](https://github.com/ustclug/Yuki/actions/workflows/pr-presubmit-checks.yml/badge.svg)](https://github.com/ustclug/Yuki/actions/workflows/pr-presubmit-checks.yml)
[![Go Report](https://goreportcard.com/badge/github.com/ustclug/Yuki)](https://goreportcard.com/report/github.com/ustclug/Yuki)

- [Requirements](#requirements)
- [Quickstart](#quickstart)
- [Handbook](#handbook)

Sync local repositories with remote.

## Requirements

* Docker
* SQLite

## Quickstart

Download the binaries from the [Release](https://github.com/ustclug/Yuki/releases) page, for example:

```bash
wget https://github.com/ustclug/Yuki/releases/download/v0.2.2/yuki-v0.2.2-linux-amd64.tar.gz
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
curl 'https://raw.githubusercontent.com/ustclug/Yuki/master/deploy/yukid.service' | sudo tee /etc/systemd/system/yukid.service
systemctl enable yukid
systemctl start yukid
systemctl status yukid
```

Setup repository:

```bash
# The repository directory must be created in advance
mkdir /tmp/repos/docker-ce

# Sync docker-ce repository from rsync.mirrors.ustc.edu.cn
cat <<EOF > /tmp/repo-configs/docker-ce.yaml
name: docker-ce
# every 1 hour
cron: "0 * * * *"
storageDir: /tmp/repos/docker-ce
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
