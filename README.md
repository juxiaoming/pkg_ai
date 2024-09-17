# pkg_ai
各平台大模型api请求封装,统一参数调用
### 安装
```
go get github.com/juxiaoming/pkg_ai
```

### 已接入大模型列表
<table>
    <tr><th>LOGO</th><th>模型名称</th><th>参考文档</th><th>应用申请</th></tr>
    <tr>
        <td><img src="./logo/moonshot.png" height="30" title="月之暗面"></td>
        <td>月之暗面</td>
        <td><a target="_blank" href="https://platform.moonshot.cn/docs/api/chat#%E5%AD%97%E6%AE%B5%E8%AF%B4%E6%98%8E">参考文档</a></td>
        <td><a target="_blank" href="https://platform.moonshot.cn/console/api-keys">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="./logo/minimaxi.png" height="30" title="minimax"></td>
        <td>Minimax</td>
        <td><a target="_blank" href="https://platform.minimaxi.com/document/ChatCompletion%20v2?key=66701d281d57f38758d581d0">参考文档</a></td>
        <td><a target="_blank" href="https://platform.minimaxi.com/user-center/basic-information/interface-key">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="./logo/volcengine.png" height="30" title="火山引擎"></td>
        <td>火山引擎</td>
        <td><a target="_blank" href="https://www.volcengine.com/docs/82379/1298454">参考文档</a></td>
        <td><a target="_blank" href="https://console.volcengine.com/ark/region:ark+cn-beijing/apiKey">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="./logo/baidu.png" height="30" title="百度千帆"></td>
        <td>百度千帆</td>
        <td><a target="_blank" href="https://cloud.baidu.com/doc/WENXINWORKSHOP/s/clntwmv7t#http%E8%B0%83%E7%94%A8">参考文档</a></td>
        <td><a target="_blank" href="https://console.bce.baidu.com/qianfan/ais/console/applicationConsole/application/v1">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="./logo/aliyun.png" height="30" title="通义千问"></td>
        <td>通义千问</td>
        <td><a target="_blank" href="https://help.aliyun.com/zh/dashscope/developer-reference/use-qwen">参考文档</a></td>
        <td><a target="_blank" href="https://dashscope.console.aliyun.com/apiKey">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="./logo/tencent.png" height="30" title="混元大模型"></td>
        <td>混元大模型</td>
        <td><a target="_blank" href="https://cloud.tencent.com/document/api/1729/105701">参考文档</a></td>
        <td><a target="_blank" href="https://console.cloud.tencent.com/cam">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="./logo/bigmodel.png" height="30" title="智谱清言"></td>
        <td>智谱清言</td>
        <td><a target="_blank" href="https://open.bigmodel.cn/dev/api/normal-model/glm-4">参考文档</a></td>
        <td><a target="_blank" href="https://bigmodel.cn/usercenter/auth">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="./logo/xfyun.png" height="30" title="科大讯飞"></td>
        <td>科大讯飞</td>
        <td><a target="_blank" href="https://www.xfyun.cn/doc/spark/HTTP%E8%B0%83%E7%94%A8%E6%96%87%E6%A1%A3.html">参考文档</a></td>
        <td><a target="_blank" href="https://console.xfyun.cn/services/bm3">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="./logo/baichuan.png" height="30" title="百川智能"></td>
        <td>百川智能</td>
        <td><a target="_blank" href="https://platform.baichuan-ai.com/docs/api">参考文档</a></td>
        <td><a target="_blank" href="https://platform.baichuan-ai.com/console/apikey">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="./logo/sensenova.png" height="30" title="商汤日日新"></td>
        <td>商汤日日新</td>
        <td><a target="_blank" href="https://platform.sensenova.cn/doc?path=/chat/ChatCompletions/ChatCompletions.md">参考文档</a></td>
        <td><a target="_blank" href="https://console.sensecore.cn/iam/Security/access-key">应用申请</a></td>
    </tr>
</table>

### 使用

#### 初始化配置
```go
// 初始化单服务配置
pkg_ai.Init(pkg_ai.NewMoonshotConf("request_url" , "sk-your_key"))

// 初始化多服务配置
pkg_ai.Init(&pkg_ai.Config{...})
```
#### 实例化服务
```go
server, err := pkg_ai.NewServer(pkg_ai.ImplementMoonshot)
if err != nil {
    fmt.Println("服务初始化失败", err)
    return
}
```
#### 常规请求参数
```go
requestData := pkg_ai.RequestData{
    Model:     "moonshot-v1-8k",
    UserQuery: "帮我写出岳飞的满江红",
}
```
#### 阻塞式请求
```go
// 常规请求
res, err := server.Chat(data)

// 自定义请求参数
res , err := server.CustomizeChat([]byte("{....}"))
```
#### 流式请求
```go
msgChan, errChan := make(chan string, 10000), make(chan error)
go func() {
    // 常规请求
    res , err := server.ChatStream(data, msgChan, errChan)

    // 自定义请求参数
    res , err := server.CustomizeChatStream([]byte("{....}"), msgChan, errChan)
}()

for {
    select {
    //todo 可引入stopChan控制请求流程
		
    case err := <-errChan:
        fmt.Println("发生错误了:", err)
        time.Sleep(time.Second * 10)
        return

    case data, ok := <-msgChan:
        if !ok {
            fmt.Println("消息管道关闭了")
            time.Sleep(time.Second * 10)
            return
        }
        if len(data) == 0 {
            continue
        }

        fmt.Println("收到数据:", data)
    }
}
```
#### 响应数据
```go
// 请求头
fmt.Println(string(res.RequestHeader))

// 请求体
fmt.Println(string(res.RequestBody))

// 响应体:
for _, item := range res.ResponseData {
    fmt.Println("响应数据:" , string(item))
}

// 提示词消耗token数量
fmt.Println(res.PromptTokens)

// 响应消耗token数量
fmt.Println(res.CompletionTokens)

// 整理后的响应数据
fmt.Println(res.ResponseText)
```
#### 模型供应商
```go
fmt.Println(server.Supplier())
```
### 建议
建议初始化配置文件之后单次调用pkg_login.Init()方法注册服务配置
### 更多
如果有好的ai模型建议,请联系我!