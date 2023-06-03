/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package jvm

import (
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"math"
	"strings"
	"time"
)

type (
	internalState struct {
		ByPid map[string]*pidJvmState
	}
	pidJvmState struct {
		RawMetrics map[string]interface{}
	}
)

var accuMetrics map[string]bool
var timeMetrics = map[string]bool{
	"ygc_time":            true,
	"fgc_time":            true,
	"safepoint_time":      true,
	"safepoint_sync_time": true,
	"application_time":    true,
}

func init() {
	accuMetrics = make(map[string]bool)
	accuMetrics["ygc_count"] = true
	accuMetrics["ygc_time"] = true
	accuMetrics["fgc_count"] = true
	accuMetrics["fgc_time"] = true
	accuMetrics["thread_started"] = true
	accuMetrics["safepoints"] = true
	accuMetrics["safepoint_time"] = true
	accuMetrics["safepoint_sync_time"] = true
	accuMetrics["application_time"] = true
	accuMetrics["parks"] = true
	accuMetrics["notifications"] = true
	accuMetrics["inflations"] = true
	accuMetrics["futile_wakeups"] = true
	accuMetrics["deflations"] = true
	accuMetrics["contended_lock_attempts"] = true
}

func addJvmMetricsFromProcess(javaProcess *criutils.ProcessInfo, rawMetrics map[string]interface{}, tags map[string]string) {
	maxMetaspaceSize := int64(0)
	reservedCodeCacheSize := int64(0)

	if javaProcess != nil {
		for _, e := range javaProcess.CmdlineSlice {
			// -XX:MaxMetaspaceSize=512m
			// -XX:ReservedCodeCacheSize=512m
			if strings.HasPrefix(e, "-XX:MaxMetaspaceSize=") {
				valueStr := e[len("-XX:MaxMetaspaceSize="):]
				maxMetaspaceSize = convertToBytes(valueStr)
			}
			if strings.HasPrefix(e, "-XX:ReservedCodeCacheSize=") {
				valueStr := e[len("-XX:ReservedCodeCacheSize="):]
				reservedCodeCacheSize = convertToBytes(valueStr)
			}
		}
		tags["user"] = javaProcess.User
	}

	// 2
	rawMetrics["meta_max"] = float64(maxMetaspaceSize)
	rawMetrics["reserved_code_cache_size"] = float64(reservedCodeCacheSize)
}

func addJvmMetrics(perfData map[string]interface{}, rawMetrics map[string]interface{}) {
	// 6
	addFloat64(perfData, rawMetrics, "sun.gc.generation.0.space.0.used", "eden_used")
	addFloat64(perfData, rawMetrics, "sun.gc.generation.0.space.0.capacity", "eden_capacity")
	addFloat64(perfData, rawMetrics, "sun.gc.generation.0.space.0.maxCapacity", "eden_max")
	addFloat64(perfData, rawMetrics, "sun.gc.generation.1.space.0.used", "old_used")
	addFloat64(perfData, rawMetrics, "sun.gc.generation.1.space.0.capacity", "old_capacity")
	addFloat64(perfData, rawMetrics, "sun.gc.generation.1.space.0.maxCapacity", "old_max")

	// 5
	addFloat64(perfData, rawMetrics, "sun.gc.metaspace.used", "meta_used")
	addFloat64(perfData, rawMetrics, "sun.gc.metaspace.capacity", "meta_capacity")
	addFloat64(perfData, rawMetrics, "sun.gc.metaspace.maxCapacity", "meta_max")
	addFloat64(perfData, rawMetrics, "sun.gc.compressedclassspace.used", "compressedclass_used")
	addFloat64(perfData, rawMetrics, "sun.gc.compressedclassspace.capacity", "compressedclass_capacity")

	// 3
	addFloat64(perfData, rawMetrics, "sun.gc.generation.2.space.0.used", "perm_used")
	addFloat64(perfData, rawMetrics, "sun.gc.generation.2.space.0.capacity", "perm_capacity")
	addFloat64(perfData, rawMetrics, "sun.gc.generation.2.space.0.maxCapacity", "perm_max")

	// GC等相关的核心指标
	// 4
	addFloat64(perfData, rawMetrics, "sun.gc.collector.0.invocations", "ygc_count")
	addFloat64(perfData, rawMetrics, "sun.gc.collector.0.time", "ygc_time")
	addFloat64(perfData, rawMetrics, "sun.gc.collector.1.invocations", "fgc_count")
	addFloat64(perfData, rawMetrics, "sun.gc.collector.1.time", "fgc_time")

	// 线程相关的指标
	// 4
	addFloat64(perfData, rawMetrics, "java.threads.started", "thread_started")
	addFloat64(perfData, rawMetrics, "java.threads.live", "thread_live")
	addFloat64(perfData, rawMetrics, "java.threads.daemon", "thread_daemon")
	addFloat64(perfData, rawMetrics, "java.threads.livePeak", "thread_live_peak")

	// 锁和等待相关的指标
	// 11
	addFloat64(perfData, rawMetrics, "sun.rt.safepoints", "safepoints")
	addFloat64(perfData, rawMetrics, "sun.rt.safepointTime", "safepoint_time")
	addFloat64(perfData, rawMetrics, "sun.rt.safepointSyncTime", "safepoint_sync_time")
	addFloat64(perfData, rawMetrics, "sun.rt.applicationTime", "application_time")
	addFloat64(perfData, rawMetrics, "sun.rt._sync_Parks", "parks")
	addFloat64(perfData, rawMetrics, "sun.rt._sync_Notifications", "notifications")
	addFloat64(perfData, rawMetrics, "sun.rt._sync_MonExtant", "mon_extant")
	addFloat64(perfData, rawMetrics, "sun.rt._sync_Inflations", "inflations")
	addFloat64(perfData, rawMetrics, "sun.rt._sync_FutileWakeups", "futile_wakeups")
	addFloat64(perfData, rawMetrics, "sun.rt._sync_Deflations", "deflations")
	addFloat64(perfData, rawMetrics, "sun.rt._sync_ContendedLockAttempts", "contended_lock_attempts")

	// TLAB相关的指标，暂时先只做一个，有点多，还不知道什么意思
	// 1
	addFloat64(perfData, rawMetrics, "sun.gc.tlab.alloc", "tlab_alloc")

	// classloader相关
	// 4
	addFloat64(perfData, rawMetrics, "java.cls.loadedClasses", "loaded_classes")
	addFloat64(perfData, rawMetrics, "java.cls.unloadedClasses", "unloaded_classes")
	addFloat64(perfData, rawMetrics, "java.cls.sharedLoadedClasses", "shared_loaded_classes")
	addFloat64(perfData, rawMetrics, "java.cls.sharedUnloadedClasses", "shared_unloaded_classes")

	// code cache相关
	// 2
	addFloat64(perfData, rawMetrics, "sun.ci.nmethodCodeSize", "nmethod_code_size")
	addFloat64(perfData, rawMetrics, "sun.ci.nmethodSize", "nmethod_size")
}

func calcFinalMetrics(rawMetrics map[string]interface{}, lastPidState *pidJvmState) map[string]interface{} {
	finalMetrics := make(map[string]interface{}, len(rawMetrics))
	for k, v := range rawMetrics {
		finalMetrics[k] = v
	}

	if lastPidState != nil {
		for key := range accuMetrics {
			if f64, err := cast.ToFloat64E(lastPidState.RawMetrics[key]); err == nil {
				v := math.Max(finalMetrics[key].(float64)-f64, 0)

				if _, ok := timeMetrics[key]; ok {
					v /= float64(time.Second.Nanoseconds())
				}

				finalMetrics[key] = v
			} else {
				// 否则要清掉这些指标
				delete(finalMetrics, key)
			}
		}
	} else {
		// 如果没有老状态那么要清掉这些指标
		for key := range accuMetrics {
			delete(finalMetrics, key)
		}
	}

	return finalMetrics
}

func addFloat64(from, to map[string]interface{}, fromKey, toKey string) {
	if i, ok := from[fromKey]; ok {
		if f64, err := cast.ToFloat64E(i); err == nil {
			to[toKey] = f64
		}
	}
}
