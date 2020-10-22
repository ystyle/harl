### harl
> 鸿蒙应用开发实机热更新工具

### 依赖
- nfs v3
  - [配置过程](https://openharmony.gitee.com/openharmony/docs/blob/master/kernel/NFS.md)
  - [配置过程可能遇到的问题](https://openharmony.gitee.com/openharmony/docs/issues/I1YIBO)
- serial(串口)
- dev tools (可以从鸿蒙编译目录`/out/ipcamera_hi3516dv300/dev_tools/bin`复制到nfs挂载目录)
  - aa (管理APP启动，关闭)
  - bm (安装卸载APP)

### USAGE

```shell
E:\Code\Go\harl>harl.exe -h
NAME:
   harl - Open Harmony OS APP auto reload tool

USAGE:
   harl.exe [global options] command [command options] [arguments...]

VERSION:
   v0.0.1

COMMANDS:
   init, i   init .harm.yml
   watch, w  watch and reload app
   help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help (default: false)
   --version, -v  print the version (default: false)
```
- [下载应用](https://gitee.com/ystyle/harl/releases)
- `cd /your-project-dir` 切换到项目目录
- `harl init` 初始化配置文件
- `harl w` 监听项目文件修改

### 配置文件
>.harl.yaml
```yaml
build:
  buildtype: smartVision # 项目编译类型，会自动生成
  excludes: # 排除监听的目录
  - .gradle
  - .idea
  - gradle
  - entry/build
  - entry/node_modules
  includes: # 监听的文件类型，不能为空
  - .css
  - .hml
  - .js
  - .hap
  - .json
  nfsdir: Y:/dde # nfs 的挂载目录
  delay: 100 # 监听频率，单位ms
reload:
  dir: /nfs/dde # nfs 在开发板上的目录
  com: COM5 # 串口号
  bundlename: top.ystyle.jianmu # 项目id，init时自动生成
  abilityname: default # 项目起始界面，init时自动生成
```

### 支持情况
- 只支持windows
- 只测试过liteWearable项目改的smartVision
- 只在hi3516d上测试过