# yukictl

### Table of Content

+ [Introduction](#introduction)
+ [Handbook](#handbook)
  - [自动补全](#自动补全)
  - [获取同步状态](#获取同步状态)
  - [手动开始同步任务](#手动开始同步任务)
  - [更新仓库同步配置](#更新仓库同步配置)
  - [获取同步日志](#获取同步日志)

### Introduction

yuki 的命令行客户端。

### Handbook

#### 自动补全

```bash
# Zsh:
$ yukictl completion zsh

# Bash:
$ yukictl completion bash
```

#### 获取同步状态

```bash
$ yukictl meta ls [repo]
```

#### 手动开始同步任务

```bash
$ yukictl sync <repo>
```

开启同步任务的 debug 模式，并查看同步日志
```bash
$ yukictl sync --debug <repo>
```

#### 更新仓库同步配置

新增或修改完仓库的 YAML 配置后，需要执行下面的命令来更新配置。
```bash
$ yukictl reload <repo>
```
注意：在新增配置前需要先创建仓库相应的 `storageDir`。

如果不带任何参数的话，则该命令会更新所有仓库的同步配置，并且删除配置里没有但数据库里有的仓库配置。
```bash
$ yukictl reload
```

若需要删除仓库，则可以删除相应的配置文件然后执行 `yukictl repo rm <repo>` 或直接 `yukictl reload` 来从数据库里删除配置。

#### 获取同步日志

当前的同步日志:
```bash
$ yukictl ct logs <container ID or repo name>
```

跟踪并从倒数第 5 行开始:
```bash
$ yukictl ct logs --tail 5 -f <container ID or repo name>
```

列出以往的同步日志:
```bash
$ yukictl repo logs --stats <repo name>
```

查看以往倒数第二次同步日志的倒数 10 行:
```bash
$ yukictl repo logs -n 1 --tail 10 <repo name>
```
