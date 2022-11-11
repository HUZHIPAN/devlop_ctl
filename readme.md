[TOC]

# 部署工具使用说明文档

### 概要：

​		部署管理工具（`lwctl`）用于管理WEB服务，针对乐维WEB的环境部署及持续可靠的版本更新支持，使用`lwctl`工具进行部署环境的管理，保证环境的一致性，简化部署及更新操作，`lwctl`可应用在所有`linux`系统上，且本身无任何系统库、软件依赖，即使在最小安装的`linux`下也可运行

### 前置要求：

#### 使用门槛：

​	1、懂`linux` 基本操作即可（不懂在指导下也可完成）

#### 系统要求：

1. 使用部署工具需要`root`权限
2. 需要指定一个部署目录，不指定默认使用`/itops`（推荐）

### 使用场景：

 #### 一、生产环境快速构建（部署）

​		若无其它限制，部署一个标准产品需要如下步骤（不修改访问端口、部署目录的情况下）

​		1、应用部署包

​				`lwctl apply -f deploy_package_6.2.tar.gz `

​		2、配置运行环境

​				--mac-addr：指定容器内的网卡物理地址，填写申请的license授权码绑定的物理地址

​				--web-api-gateway：提供后端服务的地址，格式为 http://访问IP:8081

​				`lwctl build --mac-addr=00-15-5D-A0-76-41 --web-api-gateway=http://192.168.1.1:8081`

​		3、启动WEB容器

​				`lwctl web -s start`

​		4、关于数据库配置、其他初始化访问WEB服务在页面上填写即可

#### 二、升级、更新

​		拿到开发提供的产品更新包（增量或全量），执行应用更新包命令即可，环境的配置、初始化等都自动完成。在更新镜像或站点配置或其它需要重载服务时，处在运行中的WEB容器将会自动重启。

​			`lwctl apply -f upgrade_package.tar.gz`

#### 三、管理WEB服务

​		对部署环境WEB服务的管理，包括 查看（status）、启动（start）、停止（stop）、重启（restart）等操作

​			`lwctl web -s status`

#### 四、切换版本（回滚更新）

​			1、查看更新的版本列表

​				`lwctl app -l`

​			2、切换至一个版本（需要自行判断此版本是稳定的）

​				`lwctl app -c v6.2.1`



------

> tips：使用工具，不需要关注下面的内容

# 部署工具实现说明文档

### 概要：

​		`lwctl`定义为一个部署工具，本身是无状态的。产品版本更新管理依赖的是部署目录的`git`仓库，`分支版本管理`、`增量包更新`、`代码回滚`等操作皆是使用`git`本身的特性；WEB服务的状态管理则是基于`runc`启动带命名空间的进程（容器）；`lwctl` 通过部署目录来区分操作的是哪一个环境，因此可在一个操作系统上部署多套`web服务`，且彼此之间相互独立，在启动时指定不同服务端口即可。

### 部署目录结构：

​	使用`lwctl`工具管理的部署目录结构如下：

```
├── data								web运行时持久化目录，容器启动时按每个子目录和文件按结构软链接到lwjk_app下
│   ├── .env							web应用的环境参数配置，初始化时生成
│   ├── .license						license授权文件
│   ├── environments
│   └── web
│       ├── app
│       │   └── config.json				前端生成的配置文件
│       ├── config
│       ├── uploads
│       ├── z							zabbix原⽣的`ui`代码包下的内容放到此目录，需要调整默认的zabbix版本时，可手动替换此目录下内容
│       └── zbx
├── deployment							部署管理工具操作的配置及代码目录
│   ├── .env							lwctl build 指定参数时生成的环境配置
│   ├── etc								配置包目录（更新配置包时操作此目录）
│   │   ├── nginx
│   │   │   ├── nginx.conf				nginx配置文件
│   │   │   └── vhosts					nginx站点配置目录
│   │   ├── php
│   │   │   ├── ext.d					php扩展目录，容器启动时会软链接到php扩展安装目录
│   │   │   ├── php-fpm.conf			php-fpm配置文件
│   │   │   └── php.ini					php配置文件
│   │   ├── start.sh					WEB容器启动命令
│   │   └── web
│   │       └── init.sh					web初始化命令
│   ├── logs
│   │   ├── exec						容器内执行脚本、命令日志
│   │   └── nginx						nginx启动产生的日志
│   └── lwjk_app						web项目主目录（产品全量包、增量包操作此目录）
├── tmp
     ├── lock.pid						互斥锁，记录当前正在操作此部署目录进程pid
     ├── logs							工具操作记录日志（更新日志、操作审计等）
     ├── unpack							应用更新包时临时解压目录
     └── upload_packages			    serviced上传更新包目录
```

### 部署目录权限：

- 使用普通用户权限运行`lwctl`部署时，部署目录的所有者为当前用户
- 使用`root`权限运行`lwctl`工具时，会将部署目录的所有者赋予`itops`用户（不存在则创建）

### 命令说明：

​			解释命令做的操作及实现方式

### `lwctl apply` 	

##### 应用更新包

​		应用更新包操作会解析`package.tar.gz`更新包文件中`config.yaml`配置文件的更新操作，`config.yaml`文件格式如下，其中`components`中的各项可任选或组合使用，`commands`为更新后执行的命令。`config.yaml`示例如下：

```yaml
metadata:
  actionType : apply
  description : 初始化部署包
  commands : ["php bin/manager init","php bin/manager module/i dashboard"] # 更新完后执行命令
components:
  - name : "v6.1.0" # 版本号
    type : product
    desc : "产品6.1.0全量发布包"
    package: ./lwjk_app.tar.gz # 在产品包在更新包内的相对路径
  - name : "v6.1.0" # 版本号
    type : feature
    desc : "增量更新zabbix的ui目录到web/z"
    package: ./feature.tar.gz
  - name : "lwapp_image_build:0.5" # 镜像的tag
    type : image
    desc : "镜像部署"
    package: ./lwapp_image_build.tar
  - name : "v0.1" # 版本号或唯一标识（凭此回滚更新）
    type : configure
    desc : "配置更新包"
    package: ./etc.tar.gz
```

如果当前更新存在全量版本更新包时，会自动重启容器，执行初始化命令；

如果当前更新存在镜像包或配置包 并且 WEB容器正在运行中，会自动按照之前的环境配置重新生成运行容器，并重启容器；

如果当前环境WEB容器正在运行中，会在更新完成后执行`commands`中配置的命令；

​		更新包类型说明：

​			1、`product`  产品版本更新包，更新`lwjk_app`目录，每次更新为全量更新。`lwjk_app`目录使用`git`进行版本管理，更新时以`full_版本号`命名创建一个分支并切换至次分支，然后在`lwjk_app`目录下解压产品代码包，之后提交代码。

​			2、`feature` 增量更新包，操作`lwjk_app`目录，若增量包版本号发生变化，会已`incr_版本号`命名创建一个分支并切换至此分支，否则直接在产品版本包分支进行操作，在`lwjk_app`目录下解压增量更新包，提交代码。

​			3、`image` 镜像更新包，将`config.yaml`中配置的镜像load到当前环境，重新打标签tag为`lwapp_image_web:$version`，$version为镜像版本号，当前环境每次更新镜像，镜像版本号会自增。

​			4、`configure` 配置更新包，操作`etc`目录，使用`git`版本管理，以每个配置包版本号创建分支。在生成运行容器时，切换到配置包版本对应的`_active`分支，`_active`后缀为临时分支，代表配置包已经过预处理。

在`product`或`feature`类型更新时，会检查`lwjk_app`下忽略版本管理的文件夹软链接不存在则创建链接至对应持久化目录文件夹，忽略管理的目录及文件如下：

```apl
// 忽略目录
	"/environments",
	"/runtime",
	"/web/config",
	"/web/uploads",
	"/web/z",
	"/web/zbx",
	"/web/assets",
// 忽略文件
	"/.env",     // 环境配置
	"/.license", // 授权码
	"/web/app/config.json", // 前端资源配置文件
```

##### 从已部署项目加载

​		支持从原方式部署成功的`lwjk_app`目录加载代码到工具产品管理当中，如果当前系统已存在非使用`lwctl`工具部署的项目时，可直接使用`--load-path`将`lwjk_app`主项目目录中的代码及数据加载到`lwctl` 工具管理的部署目录当中，且不影响原已部署的项目，可同时存在，平滑过渡，加载仅包括`lwjk_app`项目的产品主目录，其它依赖则直接替换到使用管理工具对应的`镜像包`和`配置包`。



### `lwctl build` 

##### 生成WEB运行环境配置

​			在当前环境部署好一个环境（已应用过镜像包、配置包、产品包）时，可使用此命令生成运行环境配置，可选参数包括：

​				web访问端口`--web-port`    web页面访问的端口

​				后端服务接口`--web-api-port`  web后端接口使用的端口

​				后端接口地址`--web-api-gateway`  web后端接口访问地址，如：http://192.168.1.1:8081

​		build操作会根据指定的参数，以`lwops_web`为前缀，拼接当前部署目录路径如`/itops`将目录分隔符替换为`_`后得到的字符名称为容器名称，创建以目录区分的部署环境的运行容器，后续操作WEB的启动、停止都是操作此容器。

​		build操作会对`etc/nginx/vhosts`目录下的`nginx配置文件`进行预处理操作（宏替换），将其中的`{{$WEB_PORT}}`、`{{$WEB_API_PORT}}`等宏替换为实际指定的值；替换容器启动命令`start.sh`的`{{$UID}}`为实际执行的用户`uid`

​		容器内运行的`php-fpm`、`nginx`进程的用户的`uid`与容器外的`itops`用户`uid`相同，（在执行build时预处理配置文件时绑定），但其运行的用户名称始终是`itops`，实现将当前操作用户的`uid`与容器内的`itops`用户关联起来。这样做的目的是为了部署时不依赖`root`权限或固定某个用户，只要当前用户拥有部署目录的权限即可

​		生成容器成功后，会在部署目录写入环境配置文件，在`部署目录/deployment/.env` 保存生成容器的配置参数，后续`apply`操作更新的自动重新生成WEB运行环境配置都是基于此环境配置文件。



### `lwctl web`

##### 管理WEB服务

​		管理web服务的启动、停止，重启等服务相关的操作，本质是使用`runc`启动带命名空间的进程（WEB容器），在WEB容器启动时，会以容器内的`itops`用户执行`部署目录/itops/etc/web/init.sh`脚本，做相关的web初始化操作。

​		`/itops/etc/start.sh`为 容器的启动脚本，用于启动主要的服务进程。

###### web初始化脚本

```bash
# 注意：web初始化脚本中的命令需要保证多次执行不影响
# 删除前端资源文件配置 删除软链接指向的文件
rm -rf `readlink /itops/nginx/html/lwjk_app/web/app/config.json` 
/itops/php/bin/php /itops/nginx/html/lwjk_app/bin/manager init --choice=W --with-check=0 --web-api={{$BACKEND_API_GATEWAY}}

# 判断已经初始化过（存在.env配置）就执行命令行初始化操作
if [ -s `readlink /itops/nginx/html/lwjk_app/.env` ]; then 
    /itops/php/bin/php /itops/nginx/html/lwjk_app/bin/manager init --choice=A --with-check=0 # 执行所有初始化
else 
    /itops/php/bin/php /itops/nginx/html/lwjk_app/bin/manager init --choice=F --with-check=0 # 检查文件夹权限
fi
```

​		`php`、`nginx`以及系统依赖的库等，在镜像中已经预置好，以下是容器启动脚本，描述容器在启动时所做的操作，其中`nginx`为前台任务，保证容器不退出，请注意容器启动脚本是以容器内的`root`用户身份执行

###### 容器启动脚本

```bash
# 添加容器内的itops用户，{{$UID}}为当前操作用户的uid，关联到容器内的itops用户，容器内itops gid与uid相同
groupadd --gid {{$UID}} itops
useradd --uid {{$UID}} --gid {{$UID}} itops &> /dev/null

# 将配置包的php扩展软链到php的扩展目录
ln -sf /itops/etc/php/ext.d/* /itops/php/lib/php/extensions/no-debug-non-zts-20170718/
# 修改容器内时区
ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo 'Asia/Shanghai' >/etc/timezone
# linux计划任务服务
crond

# 启动php-fpm服务（itops用户）
/itops/php/sbin/php-fpm --fpm-config /itops/etc/php/php-fpm.conf
# 启动nginx服务（itops用户），前台运行
# （标准输出和错误输出）定向输出到指定文件
/itops/nginx/sbin/nginx -c /itops/etc/nginx/nginx.conf -g 'daemon off;' &> /itops/logs/nginx/nginx_stdout.log
```



### `lwctl app`

##### 管理产品更新版本

​		`lwjk_app ` 目录实际上是一个`git`仓库，每一个分支代表了一次版本更新，其中`full_`前缀为全量更新版本；`incr_`前缀未增量更新版本，增量更新版本是基于全量更新版本分支创建的分支，在其之上进行提交。

​		`lwctl app` 命令用于列出和回滚在部署环境更新的版本记录，使用`lwctl apply`应用更新时发送错误失败时工具会自动回滚，但更新效果未达预期时通常不能识别到，如果需要回滚到某个稳定版本，可使用此命令进行切换。在进行版本回滚时，会自动重启容器并执行web初始化脚本。

​		当前执行的操作会涉及`lwjk_app`目录（更新和切版本回滚）时，如果目录存在未忽略未提交的变动时（手动修改了代码或WEB程序修改了未忽略目录之外的文件），会先在当前分支将变动提交，在执行后续的操作，保证不丢失改动。



### `lwctl etc`

##### 管理配置包版本

​	`lwctl etc` 命令用于列出和切换当前部署环境 配置包的更新版本

​		配置包目录操作`部署目录/deployment/etc`目录，`etc`目录使用`git`进行版本管理，使用配置包的版本号创建分支，每个分支对应一个配置包版本。

​		由于在运行时对配置包中部分文件进行了预处理，引入配置包版本号拼接`_active`后缀的命名规则，表示当前应用的是哪个版本的配置包，`_active`表示当前分支已经经过预处理，其中包含的宏已经被替换，可以被容器内的程序使用。

​		在进行配置包版本切换时，会先停止WEB容器，再根据当前环境`部署目录/deployment/.env ` 文件重新生成运行容器后启动，执行web初始化脚本。

​		当前执行的操作会涉及`etc`目录（更新和切配置包版本）时，如果目录存在未忽略未提交的变动时（手动修改了配置），会先在当前分支将变动提交，在执行后续的操作，保证不丢失改动。



### `lwctl exec`

##### 在WEB容器内执行命令

​		用于在当前部署目录运行的WEB容器中执行命令，使用容器内的`itops`用户权限（对应容器外的`itops`用户），执行命令的工作目录为容器内的`/itops/nginx/html/lwjk_app`目录下，容器中的常用工具及环境变量已经内置好。

​		执行完成可在`部署目录/deployment/logs/exec`目录下查看命令执行日志和结果。

​		

### 其他：

```
其他维护或操作可查看下列支持的命令及子命令（可使用 lwctl 子命令 --help 获取使用帮助）：

commands：
apply      应用包更新（部署或更新）              示例：lwctl apply -f package.tar.gz        
build      创建运行容器，配置访问端口等        示例：lwctl build --help                   
app        管理产品版本                        示例：lwctl app -c v6.0.1                  
web        管理WEB服务的启动停止                示例：lwctl web --help                     
etc        管理配置包版本                      示例：lwctl etc --help                     
exec       在运行的WEB容器内执行命令             示例：lwctl exec -c 'php bin/manager init' 
serviced   启动http服务接收更新任务              示例：lwctl serviced -daemon -P 8082       
```

