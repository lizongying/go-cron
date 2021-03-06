# go simple cron

### dev

```
go run main.go -c dev.yml
```

### build

linux

```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o cron
```

mac

```
go build -o cron
```

### run

```
./cron -c example.yml
```

### others

* 支持但不鼓励分钟以下定时任务。这种情形比较少见，在脚本中直接处理更加方便，也能与crontab保持兼容。
* 支持当定时任务上次没有结束时不再启动新进程，适用于大多数情况。默认不可修改。
* 不再支持多进程执行。这种情形比较少见，在脚本中直接处理更加方便。
* 暂时不支持暂停/重启等操作。简单才是硬道理。
* 支持分组定时任务(group)，如服务器名。
* 支持执行目录的配置。
* 暂时使用文件进行管理。好处是存储安全，简单，通过版本控制能保存历史，也有审核过程。缺点是在一些环境中上线复杂。
* 定时任务更新时间(interval)，单位分钟，默认一分钟。

