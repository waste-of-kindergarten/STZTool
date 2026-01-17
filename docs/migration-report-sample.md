# ZimaOS 数据迁移报告（示例生成模板）

**项目名称**：Synology → ZimaOS 数据迁移  
**迁移工具版本**：v0.1.0  
**操作人员**：项诗豪 
**迁移日期**：2026-01-17

---

## 1. 项目概述

- 本次迁移目标：将用户数据从群辉（Synology DSM）迁移至 ZimaOS，保持**文件结构、权限及数据完整性**。  

- 迁移类型：[分步迁移]  
- 迁移范围：群晖的全部用户数据文件(不包含应用和系统文件)

---

## 2. 迁移环境

| 环境 | 说明 |
|------|------|
| 源系统 | Synology DS3622 |
| 目标系统 | ZimaOS |
| 网络环境 | 千兆局域网 |
| 数据量 | 14.83 GB |
| 操作系统 | ubuntu  |


---

## 3. 迁移范围

### 迁移路径

- `/volume1/folder1`: 

    + `folder_name_with_s p a c e` : 带有空格的文件夹名测试
    + `video.MP4`: 2.1 GB 大文件

- `/volume1/folder2`: 

    + `complex_permissions_folder` : 带有复杂权限的文件夹测试
    + `complex_permissions_folder/...MP4` : 一个7.97GB的录制视频

- `/volume1/homes` : 各个用户的home文件夹

- `/volume2/folder3` : 另一个卷, 内含一个 4.67GB的录制视频


## 迁移用户

|组|用户|
|---|---|
|users| shihao, test1, g1test|
| testg | g1test |


### 权限迁移

- `/volume1/folder1` : 
    + shihao : Read/Write
    + test1 : Read Only 

- `/volume1/folder2` : 
    + testg : No Access (该组限制将会取消g1test的权限)
    + g1test: Read/Write 

- `/volume1/homes` : 按照各用户home权限定义

- `/volume2/folder3` : 
    + g1test : Read/Write 
    + shihao : Read/Write
    + test1 : Read/Write

---

## 4. 迁移策略

- 获取群晖用户和组
- 映射群晖用户到ZimaOS用户
- 配置路径映射 (卷 -> 硬盘)
- 文件递归扫描与上传 
    + 文件夹: 提取用户权限, 在ZimaOS创建相应文件夹并配置权限
    + 文件: 获取群晖文件句柄, 上传ZimaOS对应位置
- 异常处理：异常日志列表   


## 5. 迁移执行结果


| 指标 | 数值 |
| --- | --- | 
| 总文件夹数 | 10 |
| 总文件数 | 3 | 
| 总大小 | 14.83 GB | 
| 总耗时 | 2m 21s | 
| 平均传输速度 | 107.67 MB/s | 

## 6. 结论

- 本次迁移涉及了多卷, 多用户, 多组的权限映射和多个大文件传输, 具有相当的代表性
- 数据能够完整传输, 本次测试无失败文件
- 权限映射验证通过, 未发现越权或权限丢失问题
- 整体迁移过程较为稳定