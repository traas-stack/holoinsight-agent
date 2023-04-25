/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package timebinarysearch

// 前提:
// 1. 日志时间戳非递减
// 2. 几乎每行都有日志时间戳, 允许存在一些行解不出时间戳, 但这样行不能太多
// 3. 允许自定义时间戳解析方式

// 时间二分搜索
// 1. 读首行: 行首可能非法, 可以找前n行的第一个合法行
// 2. 读尾行: 行尾可能非法, 可以找后n行的第一个合法行
// 3. 二分法算出一个offset, 从该offset向前找到第一个最近的行
// 4. 重复流程
// 5. 找出>=指定时间的第一行
// 6. 找出<指定时间的最后一行
//
//func search(f *os.File, ts int64) {
//	// f.Name()
//}
