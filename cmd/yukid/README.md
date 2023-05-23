# yukid

### Table of Content

- [Introduction](#introduction)
- [Server Configuration](#server-configuration)
- [Repo Configuration](#repo-configuration)

### Introduction

yukid 是 yuki 的服务端，负责定期同步仓库，并且提供 RESTful API 用于管理。

### Server Configuration

yukid 的配置，路径 `/etc/yuki/daemon.toml`

```toml
## 设置 debug 为 true 后会打开 echo web 框架的 debug 模式
## 以及在日志里输出程序里打印日志的位置
#debug = true

## 设置 MongoDB 地址
## 完整格式为
## [mongodb://][user:pass@]host1[:port1][,host2[:port2],...][/database][?options]
#db_url = "127.0.0.1:27017"

## 设置 db 名字
#db_name = "mirror"

## 数据所在位置的文件系统
## 可选的值为 "zfs" | "xfs" | "default"
## 影响获取仓库大小的方式，如果是 "default" 的话仓库大小恒为 `-1`
fs = "default"

## 每个仓库的同步配置存放的文件夹
## 每个配置的后缀名必须是 `.yaml`
## 配置的格式参考下方 Repo Configuration
repo_config_dir = ["/path/to/config-dir"]

## 设置 Docker Daemon 地址
## unix local socket: unix:///var/run/docker.sock
## tcp: tcp://127.0.0.1:2375
#docker_endpoint = "unix:///var/run/docker.sock"

## 设置同步程序的运行时的 uid 跟 gid，会影响仓库文件的 uid 跟 gid
## 格式为 uid:gid
#owner = "1000:1000"

## 设置日志所在文件夹
#log_dir = "/var/log/yuki/"

## 设置 log level
## 可选的值为 "debug" | "info" | "warn" | "error"
#log_level = "info"

## 设置监听地址
#listen_addr = "127.0.0.1:9999"

## 设置同步仓库的时候默认绑定的 IP
#bind_ip = "1.2.3.4"

## 设置创建的 container 的名字前缀
#name_prefix = "syncing-"

## 设置同步完后执行的命令
#post_sync = ["/path/to/the/program"]

## 设置更新用到的 docker images 的频率
## 格式为 crontab
#images_upgrade_interval = "@every 1h"

## 同步超时时间，如果超过了这个时间，同步容器会被强制停止
## 支持使用 time.ParseDuration() 支持的时间格式，诸如 "10m", "1h" 等
## 如果为 0 的话则不会超时。注意修改的配置仅对新启动的同步容器生效
#sync_timeout = "48h"

## 将 seccomp profile 传递至 docker security-opt 参数，而非在 docker daemon 中指定
## 目的是放通 docker 默认拦截的 pidfd_getfd 系统调用，让 binder 强制让同步程序在特定地址上发起连接
## 留空时使用 docker daemon 默认的 seccomp 配置
#seccomp_profile = "/path/to/seccomp/profile.json"
```

### Repo Configuration

yukid 启动的时候只会从数据库里读取仓库的同步配置，不会读取 `repo_config_dir` 下的配置，所以如果有新增配置的话需要执行 `yukictl reload` 来把配置写到数据库中。

存放在 `repo_config_dir` 下的每个仓库的同步配置，文件名必须以 `.yaml` 结尾。

示例如下。不同的 image 需要的 envs 可参考 [这里](https://github.com/ustclug/ustcmirror-images#table-of-content)。

```yaml
name: bioc # required
image: ustcmirror/rsync:latest # required
interval: 2 2 31 4 * # required
storageDir: /srv/repo/bioc # required
logRotCycle: 1 # 保留多少次同步日志
bindIP: 1.2.3.4
retry: 2 # 同步失败后的重试次数
envs: # 传给同步程序的环境变量
  RSYNC_HOST: rsync.exmaple.com
  RSYNC_PATH: /
  RSYNC_RSH: ssh -i /home/mirror/.ssh/id_rsa
  RSYNC_USER: bioc-rsync
  $UPSTREAM: rsync://rsync.example.com/ # 可选变量，设置 yuki 显示的同步上游
volumes: # 同步的时候需要挂载的 volume
  # 注意: 由于 MongoDB 的限制，key 不能包含 `.`
  /etc/passwd: /etc/passwd:ro
  /ssh: /home/mirror/.ssh:ro
```

当存在多个目录时，配置将被字段级合并，同名字段 last win。举例：

daemon.yaml

```yaml
repo_config_dir = ["common/", "override/"]
```

common/centos.yaml

```yaml
name: centos
storageDir: /srv/repo/centos/
image: ustcmirror/rsync:latest
interval: 0 0 * * *
envs:
  RSYNC_HOST: msync.centos.org
  RSYNC_PATH: CentOS/
logRotCycle: 10
retry: 1
```

override/centos.yaml

```yaml
interval: 17 3-23/4 * * *
envs:
  RSYNC_MAXDELETE: "200000"
```

`yukictl repo ls centos`

```json
{
  "name": "centos",
  "interval": "17 3-23/4 * * *",
  "image": "ustcmirror/rsync:latest",
  "storageDir": "/srv/repo/centos/",
  "logRotCycle": 10,
  "retry": 2,
  "envs": {
    "RSYNC_HOST": "msync.centos.org",
    "RSYNC_MAXDELETE": "200000",
    "RSYNC_PATH": "CentOS/"
  }
}
```
