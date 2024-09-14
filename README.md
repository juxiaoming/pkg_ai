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
        <td><img src="https://platform.moonshot.cn/logo.png" height="30" title="月之暗面"></td>
        <td>月之暗面</td>
        <td><a target="_blank" href="https://platform.moonshot.cn/docs/api/chat#%E5%AD%97%E6%AE%B5%E8%AF%B4%E6%98%8E">参考文档</a></td>
        <td><a target="_blank" href="https://platform.moonshot.cn/console/api-keys">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="https://filecdn.minimax.chat/public/Group.png?x-oss-process=image/format,webp" height="30" title="minimax"></td>
        <td>Minimax</td>
        <td><a target="_blank" href="https://platform.minimaxi.com/document/ChatCompletion%20v2?key=66701d281d57f38758d581d0">参考文档</a></td>
        <td><a target="_blank" href="https://platform.minimaxi.com/user-center/basic-information/interface-key">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="https://portal.volccdn.com/obj/volcfe/logo/appbar_logo_dark.2.svg" height="30" title="火山引擎"></td>
        <td>火山引擎</td>
        <td><a target="_blank" href="https://www.volcengine.com/docs/82379/1298454">参考文档</a></td>
        <td><a target="_blank" href="https://console.volcengine.com/ark/region:ark+cn-beijing/apiKey">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="https://nlp-eb.cdn.bcebos.com/static/eb/asset/logo.8a6b508d.png" height="30" title="百度千帆"><img src="https://nlp-eb.cdn.bcebos.com/static/eb/asset/logo-name.7e54ad31.png" height="18" title="百度千帆"></td>
        <td>百度千帆</td>
        <td><a target="_blank" href="https://cloud.baidu.com/doc/WENXINWORKSHOP/s/clntwmv7t#http%E8%B0%83%E7%94%A8">参考文档</a></td>
        <td><a target="_blank" href="https://console.bce.baidu.com/qianfan/ais/console/applicationConsole/application/v1">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="https://img.alicdn.com/imgextra/i1/O1CN01AKUdEM1qP6BQVaYhT_!!6000000005487-2-tps-512-512.png" height="30" title="通义千问"></td>
        <td>通义千问</td>
        <td><a target="_blank" href="https://help.aliyun.com/zh/dashscope/developer-reference/use-qwen">参考文档</a></td>
        <td><a target="_blank" href="https://dashscope.console.aliyun.com/apiKey">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="https://cloudcache.tencent-cloud.com/qcloud/portal/kit/images/slice/logo.23996906.svg" height="30" title="混元大模型"></td>
        <td>混元大模型</td>
        <td><a target="_blank" href="https://cloud.tencent.com/document/api/1729/105701">参考文档</a></td>
        <td><a target="_blank" href="https://console.cloud.tencent.com/cam">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="https://chatglm.cn/img/icons/msapplication-icon-144x144.png" height="30" title="智谱清言"></td>
        <td>智谱清言</td>
        <td><a target="_blank" href="https://open.bigmodel.cn/dev/api/normal-model/glm-4">参考文档</a></td>
        <td><a target="_blank" href="https://bigmodel.cn/usercenter/auth">应用申请</a></td>
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
fmt.Println("请求的完整数据:", string(res.RequestData))
fmt.Println("响应的完整数据:", string(res.ResponseData))
fmt.Println("提示词消耗token数量:", res.PromptTokens)
fmt.Println("响应消耗token数量:", res.CompletionTokens)
fmt.Println("整理后的响应数据:", res.ResponseText)
```
### 建议
建议初始化配置文件之后单次调用pkg_login.Init()方法注册服务配置
### 更多
等我！