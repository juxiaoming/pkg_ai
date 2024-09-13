# pkg_ai
各平台大模型api请求封装
### 安装
```
go get github.com/juxiaoming/pkg_ai
```

### 支持第三方登录
<table>
    <tr><th>Ai大模型</th><th>参考文档</th><th>应用申请</th></tr>
    <tr>
        <td><img src="https://platform.moonshot.cn/logo.png" height="30" title="月之暗面"></td>
        <td><a target="_blank" href="https://platform.moonshot.cn/docs/api/chat#%E5%AD%97%E6%AE%B5%E8%AF%B4%E6%98%8E">参考文档</a></td>
        <td><a target="_blank" href="https://platform.moonshot.cn/console/api-keys">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="https://filecdn.minimax.chat/public/Group.png?x-oss-process=image/format,webp" height="30" title="minimax"></td>
        <td><a target="_blank" href="https://platform.minimaxi.com/document/ChatCompletion%20v2?key=66701d281d57f38758d581d0">参考文档</a></td>
        <td><a target="_blank" href="https://platform.minimaxi.com/user-center/basic-information/interface-key">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="https://portal.volccdn.com/obj/volcfe/logo/appbar_logo_dark.2.svg" height="30" title="volc"></td>
        <td><a target="_blank" href="https://www.volcengine.com/docs/82379/1298454">参考文档</a></td>
        <td><a target="_blank" href="https://console.volcengine.com/ark/region:ark+cn-beijing/apiKey">应用申请</a></td>
    </tr>
    <tr>
        <td><img src="https://bce.bdstatic.com/img/favicon.ico" height="30" title="文心一言"></td>
        <td><a target="_blank" href="https://cloud.baidu.com/doc/WENXINWORKSHOP/s/clntwmv7t#http%E8%B0%83%E7%94%A8">参考文档</a></td>
        <td><a target="_blank" href="https://console.bce.baidu.com/qianfan/ais/console/applicationConsole/application/v1">应用申请</a></td>
    </tr>
</table>

### 使用
```go
// 初始化配置
pkg_ai.Init(pkg_ai.NewMoonshotConf("request_url" , "sk-your_key"))

// 初始化多服务配置
pkg_ai.Init(&pkg_ai.Config{...})

// 初始化服务
server, err := pkg_ai.NewServer(pkg_ai.ImplementMoonshot)
if err != nil {
    fmt.Println("服务初始化失败", err)
    return
}

// 请求数据
requestData := pkg_ai.RequestData{
    Model:     "moonshot-v1-8k",
    UserQuery: "帮我写出岳飞的满江红",
}

// 阻塞式请求
fmt.Println(server.Chat(data))

// 流式请求
msgChan, errChan, stopChan := make(chan string, 10000), make(chan error), make(chan struct{})
fmt.Println(server.ChatStream(data, msgChan, errChan, stopChan))

// 自定义请求参数
fmt.Println(server.CustomizeChat([]byte("{....}")))
fmt.Println(server.CustomizeChatStream([]byte("{....}"), msgChan, errChan, stopChan))
```
### 建议
建议初始化配置文件之后单次调用pkg_login.Init()方法注册服务配置
### 更多
等我！