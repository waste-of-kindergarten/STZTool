# STZTool

Synology 到 ZimaOS 引导迁移工具(后端).

## 用户部分

### 准备

请尽量保证ZimaOS中待存放迁移数据的硬盘干净, 以妨迁移中发生冲突导致失败.

### 迁移方式

通过SSH连接Synology的管理员(可以不是超级管理员), 和ZimaOS提供的API完成文件传输和权限配置

### 权限映射

Synology 使用的是ACL进行权限管理, 包含用户, 组的权限; 在ZimaOS中并没有如此复杂的权限, 因此需要进行权限映射.

如果您想让特定的Synology的管理员用户映射到ZimaOS的管理员用户, 请使用该管理员身份登录SSH, 并保证ZimaOS的用户名同名.

本迁移工具首先收集人类用户和人类组, 对Synology中出现的文件夹, 迁移工具首先提取所有有效的人类用户/组的ACL权限, 并根据ACL的逻辑得到最终的用户权限列表.

### 迁移逻辑

在Synology允许多组硬盘和卷的存在, 而ZimaOS则没有如此复杂的管理, 因此迁移工具允许用户对卷进行映射到ZimaOS的特定硬盘上

> 注意: 虽然允许多对一迁移的情况, 但是要避免不同的卷下存在相同的文件/文件夹名称, 否则可能会产生冲突



## 开发部分

### auth 

验证有关功能, 主要用于获取Authorization以便访问ZimaOS的API

### fileutils 

与文件相关的函数, 主要函数和说明如下.

- `CreateZimaOSDir(path string) error` : 在`path`创建ZimaOS的路径

- `MapZimaOSDir(path string, acl *FolderACL) error` : 对路径映射用户权限

- `UploadFile(reader io.Reader, size int64, path string) error` : 上传文件

- `GetSynologyACL(path string) (*FolderACL, error) ` : 在群晖中读取用户对`path`路径的权限

- `GetRemoteFile(remotePath string) (io.ReadCloser, int64, error) : error` : 获取群晖文件"句柄", 以及文件大小

- `ListRemoteDir(remotePath string) ([]string, []string, error)` : 获取群晖路径下的内容, 得到文件列表和文件夹列表

### public

用于存放一些全局变量

### service

为了前端用户操作的部分, 主要获取群晖待迁移卷和ZimaOS挂载硬盘情况

### userutils

用户相关的函数.

- `SshConnect() error` : 链接群晖的Ssh

- `GetSynologyUser() error` : 获取群晖用户列表

- `GetSynologyGroup() error` : 获取群晖组列表

- `CreateZimaUser() error` : 创建ZimaOS用户

- `MapZimaUser(username string) error`: 创建ZimaOS用户的核心功能, 新创建的账户密码默认`123456`, 已经创建的账户密码不变



