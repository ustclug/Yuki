# yukid

### Table of Content
* [Introduction](#introduction)
* [Server Configuration](#server-configuration)
* [Repo Configuration](#repo-configuration)

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
repo_config_dir = "/path/to/config-dir"

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
volumes: # 同步的时候需要挂载的 volume
  # 注意: 由于 MongoDB 的限制，key 不能包含 `.`
  /etc/passwd: /etc/passwd:ro
  /ssh: /home/mirror/.ssh:ro
```
