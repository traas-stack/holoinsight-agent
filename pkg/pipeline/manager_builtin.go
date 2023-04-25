/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package pipeline

// builtinConfigPrefix is the config prefix of all builtin tasks
const builtinConfigPrefix = "BUILTIN_"

// commonSysTaskTypes are the types of common sys tasks
var commonSysTaskTypes = []string{"cpu", "mem", "load", "traffic", "tcp", "process", "swap", "disk"}

// commonSysTaskTags contain tags which need to be added into datum generated by common sys tasks
var commonSysTaskTags = []string{"hostname", "ip"}
