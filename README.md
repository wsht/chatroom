# ChatRoom

## 为什么写这个
这个项目仅仅是为了自己梳理一些关于聊天室的知识而建立的。属于个人学习内容
我希望这个项目可以帮助一些想要学习或者搭建一个聊天室，但是没有思路的同学，提供一点方向。
因为本人在没有接触过相关知识的时候，是比较迷茫的（当然本人比较菜）。

## 这个代码的质量
个人感觉比较一般，因为仅仅是搭载一个架子。这套流程起源于我们以前开发的一套游戏直播的聊天室流程。并且只提取出一部分。而且以前的聊天室是使用c语言实现的。

## 为什么选用 golang
因为这个语言相对于其他语言实现这个功能是比较简单的。当然，这个简单仅仅是对于这个简陋的架子而言

## TODO
对于这个项目而言，需要完善的东西其实挺多的，比如：
+ 配置文件的支持
+ 关键日志记录
+ 最大人数限制
+ 多房间支持
+ 等等

当然对于不同的产品肯定有不同的需求，这里仅列举一些功能

## 层级以及解释
主要包括三层，分别为后台，socket服务层，socket客户端层
+ 后台：主要记录用户的进入，发言，以及鉴权认证等功能，采用http协议和socket服务层进行交互
+ socket服务层：消息提取，转发以及和后台交互。同时 维护一个http服务，供后台调用
+ 客户端层：没什么可说的。表面意思
  
## 如何运行
首先启动后台服务
```
    go run main.go backend 
```

然后启动socket服务
```
    go run main.go server
```
单人测试
```
    go run main.go client [userid]
    go run main.go client 1
```

多人测试
```
    go run main.go multiclient [usercount]
    go run main.go multiclient 100
```