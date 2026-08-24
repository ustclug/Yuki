# yukid

### Table of Content

- [yukid](#yukid)
    - [Table of Content](#table-of-content)
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

## 设置 sqlite 数据库文件的路径
## 格式可以是文件路径或者是 url（如果需要设置特定参数的话）。例如：
## /var/run/yukid/data.db
## file:///home/fred/data.db?mode=ro&cache=private
## 参考 https://www.sqlite.org/c3ref/open.html
db_url = "/path/to/yukid.db"

## 每个仓库的同步配置存放的文件夹
## 每个配置的后缀名必须是 `.yaml`
## 配置的格式参考下方 Repo Configuration
repo_config_dir = ["/path/to/config-dir"]

## 设置同步日志存放的文件夹
## 默认值是 /var/log/yuki/
#repo_logs_dir = "/var/log/yuki/"

## 数据所在位置的文件系统
## 可选的值为 "zfs" | "xfs" | "default"
## 影响获取仓库大小的方式，如果是 "default" 的话仓库大小恒为 `-1`
## 默认值是 "default"
#fs = "default"

## 设置 Docker Daemon 地址
## unix local socket: unix:///var/run/docker.sock
## tcp: tcp://127.0.0.1:2375
## 默认值是 "unix:///var/run/docker.sock"
#docker_endpoint = "unix:///var/run/docker.sock"

## 设置同步程序的运行时的 uid 跟 gid，会影响仓库文件的 uid 跟 gid
## 格式为 uid:gid
## 默认值是 yukid 进程的 uid 跟 gid
#owner = "1000:1000"

## 设置 yukid 的日志文件
## 默认值是 "/dev/stderr"
#log_file = "/path/to/yukid.log"

## 设置 log level
## 可选的值为 "debug" | "info" | "warn" | "error"
## 默认值是 "info"
#log_level = "info"

## 设置控制面的监听端点
## 可选格式：
## host:port，例如 127.0.0.1:9999
## 绝对路径，例如 /run/yuki/yukid.sock
## 默认值是 "/run/yuki/yukid.sock"
#listen_addr = "/run/yuki/yukid.sock"

## 设置只读公开 API 的 HTTP 监听地址
## 只提供 /api/v1/metas 和 /api/v1/metas/{name}
## 默认值是 "127.0.0.1:9999"；设置为空字符串可关闭
#public_listen_addr = "127.0.0.1:9999"

## 设置同步仓库的时候默认绑定的 IP
## 默认值为空，即不绑定
#bind_ip = "1.2.3.4"

## 设置创建的 container 的名字前缀
## 默认值是 "syncing-"
#name_prefix = "syncing-"

## 设置同步完后执行的命令
## 默认值为空
#post_sync = ["/path/to/the/program"]

## 设置更新用到的 docker images 的频率
## 默认值为 "1h"
#images_upgrade_interval = "1h"

## 同步超时时间，如果超过了这个时间，同步容器会被强制停止
## 支持使用 time.ParseDuration() 支持的时间格式，诸如 "10m", "1h" 等
## 如果为 0 的话则不会超时。注意修改的配置仅对新启动的同步容器生效
## 默认值为 0
#sync_timeout = "48h"
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
bindIP: 1.2.3.4 # 同步的时候绑定的 IP，可选，默认为空；未来版本将移除
network: host # 容器所属的 docker network，可选，默认为 host
retry: 2 # 同步失败后的重试次数
envs: # 传给同步程序的环境变量
  RSYNC_HOST: rsync.exmaple.com
  RSYNC_PATH: /
  RSYNC_RSH: ssh -i /home/mirror/.ssh/id_rsa
  RSYNC_USER: bioc-rsync
  $UPSTREAM: rsync://rsync.example.com/ # 可选变量，设置 yuki 显示的同步上游
volumes: # 同步的时候需要挂载的 volume
  /etc/passwd: /etc/passwd:ro
  /home/mirror/.ssh: /home/mirror/.ssh:ro
```

当存在多个目录时，配置将被字段级合并，同名字段 last win。举例：

daemon.toml

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

### RESTful API

yukid 的完整控制面默认监听 Unix socket `/run/yuki/yukid.sock`。独立的公开 HTTP server 默认监听 `127.0.0.1:9999`，只注册 `/api/v1/metas` 和 `/api/v1/metas/{name}` 两个只读接口，可用于搭建状态页。不要将控制面 socket 直接暴露给反向代理。

可以通过 Nginx 代理公开 HTTP server：

```nginx
server {
    listen 80;
    server_name mirror-status.example.com;

    location / {
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_pass http://127.0.0.1:9999;
    }
}
```

如需直接监听外部网络，可以设置 `public_listen_addr = "0.0.0.0:9999"`，并配合防火墙限制访问。设置为空字符串会关闭公开 HTTP server。

官方 systemd unit 使用 `RuntimeDirectory=yuki`：systemd 会以服务用户创建 `/run/yuki`，并在服务停止或重启时清理该目录。手动运行或使用自定义 socket 路径时，需要自行创建可写的父目录；若异常退出后遗留 socket，应先确认没有运行中的 `yukid`，再手动删除，程序不会主动删除已有路径。

若 `listen_addr` 和 `public_listen_addr` 指向同一个 TCP 地址，yukid 会为兼容旧配置启动一个包含完整控制 API 的 listener，并输出安全警告。要获得权限隔离，应让 `listen_addr` 使用 Unix socket。

yukictl 通过完整控制面操作 yukid。
