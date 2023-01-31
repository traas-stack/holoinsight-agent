# 介绍
同步 aliyun cloudmonitor 的数据到 HoloInsight.  
实现原理就是定时轮询 aliyun openapi 查数据, 然后写到我们, 效率很低但没有其他办法.  

# 遇到的问题
1. aliyun 的 openapi 一次只能查一个 metric, 一个时间区间
2. aliyun 的 openapi 有限流, 普通用户每个月只有几百万次的免费额度; TODO 有限速吗?
3. aliyun 的 指标非常多, 我们需要精挑细选一些有用的指标
4. aliyun 的 数据 自身就有一定的延迟, 我们同步数据的时候最好按区间去同步

# 优化
1. 区间查询 + 降频
2. 减少指标
3. metrics endpoints 用 aliyun 内网域名
4. 查一下用户有没有购买对应的云产品, 如果没买就不查云产品指标了 (依赖更多阿里云api, 以及需要更高权限)
