## 编译方法

1. 到GOPATH/src下：`git clone git@gitlab.ln.ad:storage/terrasync.git`
2. 进入terrasync目录
3. 执行：`go build .` 生成二进制

## 用法

### 扫描统计
```bash
terrasync scan <uri>
```

### 迁移
```bash
terrasync migrate <uri_src> <uri_dst>
```

### 过滤条件
扫描命令支持使用`--match`和`--exclude`参数添加过滤条件，格式为`属性名 运算符 值`。

#### 支持的属性类型
1. **name**: 文件名（字符串类型）
2. **type**: 文件类型（`file` 或 `dir`）
3. **path**: 文件路径（字符串类型）
4. **size**: 文件大小（支持K, M, G, T单位，如`100`, `10K`, `2M`）
5. **modified**: 修改时间（小时为单位，如`24`表示24小时内修改的文件）

#### 支持的运算符
- `==`: 等于
- `!=`: 不等于
- `>`: 大于
- `<`: 小于
- `>=`: 大于等于
- `<=`: 小于等于
- `in`: 包含子字符串
- `like`: 模糊匹配（`%`匹配任意数量字符，`_`匹配单个字符）

#### 示例
```bash
# 扫描当前目录，排除名称包含'main'的文件
terrasync scan -exclude "name like 'main%'" .

# 扫描当前目录，只匹配大小大于10M的文件
terrasync scan -match "size > 10M" .

# 组合条件（使用and/or连接）
terrasync scan -match "type==file and size > 100K" .
```

### URI格式

1. **本地目录**: 如`/mnt/raid0/`
2. **NFS共享**: 如`192.168.22.11:/srcdir`
3. **S3桶**: 如`s3://akey:skey@192.168.22.11.bucketname/xxx`

## 使能命令行自动补全功能

```powershell
.\terrasync.exe completion powershell | Out-String|Invoke-Expression
```

## 基准测试

```powershell
go test -bench=BenchmarkConditionFilter_IsSatisfied -benchtime=10s terrasync/app/scan/...
```

## 项目目录结构

```
terrasync/                  # 项目根目录
├── .gitignore              # Git忽略文件
├── app/                    # 应用程序主目录
│   ├── migrate/            # 迁移功能模块
│   └── scan/               # 扫描功能模块
│       ├── filter.go       # 扫描filter功能代码
│       ├── report.go       # 扫描报告生成代码
│       ├── scan.go         # 扫描功能实现代码
│       ├── stat.go         # 扫描统计实现代码
│       └── utils.go        # 扫描工具函数
├── command/                # 命令行工具实现
│   ├── migrate.go          # 迁移命令实现
│   ├── scan.go             # 扫描命令实现
│   └── utils.go            # 命令工具函数
├── config.yaml             # 配置文件
├── db/                     # 数据库模块
│   ├── db.go               # 数据库接口
│   ├── factory.go          # 数据库工厂
│   └── sqlite.go           # SQLite实现
├── go.mod                  # Go模块依赖文件
├── go.sum                  # Go模块校验文件
├── jobs/                   # 任务数据目录
├── log/                    # 日志功能模块
│   └── logger.go           # 日志接口实现
├── main.go                 # 程序入口文件
├── object/                 # 对象存储接口定义
│   ├── file.go             # 文件对象实现
│   ├── interface.go        # 对象接口定义
│   ├── nfs.go              # NFS对象实现
│   └── s3.go               # S3对象实现
└── readme.md               # 项目说明文档
```