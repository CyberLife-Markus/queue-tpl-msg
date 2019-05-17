## Queue
Wechat template message push message queue.

> 注：需要先到微信公众平台添加worker端IP白名单

### 1、编译

    go build

### 2、文件配置 - Redis
修改config.yml.sample改名为config.yml，修改相关配置即可。

### 3、环境变量配置 - 微信与MySQL
增加下列环境变量，对应参数参考env.yml.sample文件

    HTTP

    DB_TYPE
    DB_PREFIX
    DB_USER
    DB_PASS
    DB_HOST
    DB_PORT
    DB_NAME
    DB_CHARSET
    
    WECHAT_APPID
    WECHAT_APPSECRET
    WECHAT_TEMPLATE
    WECHAT_REMARK


### 4、启动HTTP API

    ./queue -c config.yml httpServer

### 5、启动Worker

    ./queue -c config.yml worker

### 6、HTTP API list
---

**1\. 心跳检测**
###### 接口功能
> 检测HTTP Server是否健康，该接口不返回任何数据，只返回HTTP 200

###### URL
> [http://localhost:5000/heartbeat)

###### HTTP请求方式
> GET

###### 请求参数
> |参数|必选|类型|说明|
|:-----  |:-------|:-----|-----                               |
|-    |-    |-|-                          |

---

**2\. 推送队列**
###### 接口功能
> 发起请求后将会开始查询所有符合推送条件的数据并进行推送

###### URL
> [http://localhost:5000//pushJob)

###### HTTP请求方式
> POST FORM-DATA

###### 请求参数
> |参数|必选|类型|说明|
|:-----  |:-------|:-----|-----                               |
|type    |ture    |string|通知类型：1指定ID，2分类 |
|data    |true    |string   |数据：使用逗号分隔，如果type等于2，则此字段值填1-5|
|employer    |true    |string   |需求方名称|
|time    |true    |string   |截止时间|
|url    |true    |string   |模板消息跳转URL|

###### 返回示例
success

  
### 7、待改进
 - HTTP异步响应，减少Client耗时。√
 - 获取用户openid时增加查询缓存，提高查询效率。
 - HTTP Server增加鉴权机制，不允许白名单之外的Client访问调用。
 - 优化配置方式。
 
### 8、参考
 - 模板消息DEMO：https://github.com/GanEasy/wechatSendTemplateMessage
 - Queue-machinery：https://github.com/RichardKnop/machinery
 - ORM-GoRose：https://github.com/gohouse/gorose