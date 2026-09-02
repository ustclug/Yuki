README
=======

[![Presubmit Checks](https://github.com/ustclug/Yuki/actions/workflows/pr-presubmit-checks.yml/badge.svg)](https://github.com/ustclug/Yuki/actions/workflows/pr-presubmit-checks.yml)
[![Go Report](https://goreportcard.com/badge/github.com/ustclug/Yuki)](https://goreportcard.com/report/github.com/ustclug/Yuki)

- [Requirements](#requirements)
- [Quickstart](#quickstart)
- [Handbook](#handbook)
- [Migration Guide](#migration-guide)
- [Development](#development)

Sync local repositories with remote.

## Requirements

* Docker
* SQLite

## Quickstart

### Setup

#### Debian and Ubuntu

Download `yuki_*_amd64.deb` from the [latest release][latest-release] and install it:

  [latest-release]: https://github.com/ustclug/Yuki/releases/latest

```shell
# Using v0.6.1 for example
wget https://github.com/ustclug/Yuki/releases/download/v0.6.1/yuki_0.6.1_amd64.deb
sudo dpkg -i yuki_0.6.1_amd64.deb
```

Copy `/etc/yuki/daemon.example.toml` to `/etc/yuki/daemon.toml` and edit accordingly.

Create the `mirror` user and start the system service:

```shell
sudo useradd -m mirror
sudo systemctl enable --now yukid.service
```

#### Other distros

Download the binaries from the [latest release][latest-release]. For example:

```bash
wget https://github.com/ustclug/Yuki/releases/latest/download/yukictl_linux_amd64
wget https://github.com/ustclug/Yuki/releases/latest/download/yukid_linux_amd64

sudo cp yukictl_linux_amd64 /usr/local/bin/yukictl
sudo cp yukid_linux_amd64 /usr/local/bin/yukid
sudo chmod +x /usr/local/bin/{yukid,yukictl}
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

`yukid` and `yukictl` use `/run/yuki/yukid.sock` for the full control API by default. A separate read-only status API listens on `127.0.0.1:9999`; configure `public_listen_addr` to move or disable it. The bundled systemd unit creates and cleans `/run/yuki` through `RuntimeDirectory=yuki`.

### Configure repositories

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
mirrorz:
  - desc: Docker 软件仓库
EOF

yukictl reload
# Verify
yukictl repo ls

# Trigger synchronization immediately
yukictl sync docker-ce
```

Each sync task is mapped to a same-named logical MirrorZ repository by default.
Use `mirrorz: []` to exclude an internal task, or list one or more logical
repositories when tasks and repositories do not have a one-to-one relationship.
The public metadata API returns this mapping together with each task's raw
status and size; deployment tooling can aggregate it into a complete
`mirrorz.json`.

For more details of the configuration file, please refer to the [yukid handbook](./cmd/yukid/README.md).

## Handbook

* [yukid](./cmd/yukid/README.md): Yuki daemon
* [yukictl](./cmd/yukictl/README.md): Yuki cli

## Migration Guide

### v0.6.2 -> v0.7.0

This release changes the default control-plane endpoint and the way repository
upstream information is reported. Upgrade `yukid` and `yukictl` together; there
is no transparent rolling-upgrade order when both sides use their defaults:

* An old `yukictl` connects to the full API on `127.0.0.1:9999`, while a new
  default `yukid` exposes only the read-only metadata API there.
* A new `yukictl` connects to `/run/yuki/yukid.sock`, which an old `yukid` does
  not create.

For a normal systemd installation:

1. Stop `yukid` and upgrade both binaries or the package.
2. Install the new service unit and run `systemctl daemon-reload`. The unit uses
   `RuntimeDirectory=yuki` to create and clean `/run/yuki`.
3. Start `yukid`, then verify the control plane with `yukictl repo ls` and the
   public API with `curl http://127.0.0.1:9999/api/v1/metas`.

The new defaults are equivalent to:

```toml
# Full control API used by yukictl.
listen_addr = "/run/yuki/yukid.sock"

# Read-only /api/v1/metas endpoints for status-page consumers.
public_listen_addr = "127.0.0.1:9999"
```

Manual and containerized installations must create a writable parent directory
for the Unix socket. An unclean shutdown can leave a stale socket; verify that
no `yukid` process is running before removing it. Also ensure that users which
run `yukictl` have permission to connect to the socket. Reverse proxies should
use `public_listen_addr` instead of exposing the control socket.

As a temporary compatibility mode, explicitly setting both addresses to the
same TCP endpoint keeps the complete API on that endpoint:

```toml
listen_addr = "127.0.0.1:9999"
public_listen_addr = "127.0.0.1:9999"
```

This preserves the old access model and therefore does not isolate privileged
control APIs. Migrate clients to the Unix socket before separating the two
listeners. To use a non-default endpoint explicitly, pass it to
`yukictl --remote`; both `127.0.0.1:9999` and the legacy
`http://127.0.0.1:9999/` form are accepted.

Repository upstream detection is no longer inferred from the sync image name
and image-specific environment variables. Each repository must use one of
these mechanisms:

* Set the special `$UPSTREAM` entry explicitly in its YAML configuration:

    ```yaml
    envs:
      $UPSTREAM: rsync://rsync.example.org/module
    ```

* Have the sync container write the upstream URL to
  `/log/yuki_upstream.txt`. Yuki mounts the repository's log directory at
  `/log` and reads this file after the sync finishes.

The ordinary `UPSTREAM` key and variables such as `RSYNC_HOST`, `RSYNC_PATH`,
`APTSYNC_URL`, or rclone-specific settings are no longer interpreted by Yuki.
Until a new repository completes its first sync, its metadata may therefore
contain an empty `upstream`. Existing database values are retained when no new
upstream is provided.

Other compatibility notes:

* Repository names containing `/`, or equal to `..`, are now rejected during
  reload.
* `/api/v1/metas` responses contain a new `mirrorz` array. Update strict JSON
  schemas or consumers which reject unknown fields. Omitting `mirrorz` in a
  task configuration creates a same-name mapping; `mirrorz: []` explicitly
  excludes the task.
* `yukid` and `yukictl` now exit with status 1 when command execution fails.
* If `docker_endpoint` is omitted, `DOCKER_HOST` is honored before falling back
  to `unix:///var/run/docker.sock`. Set `docker_endpoint` explicitly when the
  service environment must not affect daemon selection.
* `bindIP` still works but is deprecated; migrate source-address selection to a
  Docker network before a future release removes it.
* Building Yuki now requires Go 1.25 or newer. Go users of
  `pkg/yukictl/factory.Factory` must also update implementations and calls for
  `RESTClient() (*resty.Client, error)`.

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

## Development

* Build `yukid`:

    ```shell
    make yukid
    ```

* Build `yukictl`:

    ```shell
    make yukictl
    ```

* Build Debian package:

    ```shell
    make deb
    ```

* Lint the whole project:

    ```shell
    make lint
    ```
