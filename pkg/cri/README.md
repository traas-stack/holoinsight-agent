# 介绍
这个包存放 容器运行时(cri) 的接口/模型/辅助函数. 子目录是一些具体的 cri 实现.

docker/ 标准docker
pouch/ alibaba pouch 

# 元数据定制
1. app 的来源: 用户自定义方式(labels/ENV) > 标准 app 标签
2. hostname 的来源: 用户自定义方式(labels/ENV) > pod.spec.Hostname
