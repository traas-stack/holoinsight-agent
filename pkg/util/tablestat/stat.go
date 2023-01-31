package tablestat

import (
	"bytes"
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"strconv"
	"sync"
	"time"
)

type (
	TableStatManger struct {
		sync.Mutex
		x        map[string]*TableEntry
		dataChan chan *AddItem
		// 一个很误差很高的now
		estimatedNow int64
		printSecond  int
		reporters    []Reporter
	}

	TableEntry struct {
		x                  map[string]int64
		dirty              bool
		lastTouchTimestamp int64
	}

	AddItem struct {
		table string
		key   string
		value int64
		sum   bool
	}

	ReportResult struct {
		TableMetrics map[string]map[string]int64
	}

	Reporter interface {
		Report(ReportResult)
	}
)

var DefaultManager = NewTableStatManger(31)

func init() {
	DefaultManager.Start()
}

func NewTableStatManger(printSecond int) *TableStatManger {
	return &TableStatManger{
		x:            make(map[string]*TableEntry),
		dataChan:     make(chan *AddItem, 65536),
		estimatedNow: util.CurrentMS(),
		printSecond:  printSecond,
	}
}

func (m *TableStatManger) RegisterReporter(r Reporter) {
	m.Lock()
	defer m.Unlock()
	m.reporters = append(m.reporters, r)
}

func (m *TableStatManger) scheduleOnce() {
	// 在31的时候才去打印
	now := time.Now()
	if now.Second() < m.printSecond {
		time.AfterFunc(time.Duration(m.printSecond-now.Second())*time.Second, m.printOnce)
	} else {
		time.AfterFunc(time.Duration(60+m.printSecond-now.Second())*time.Second, m.printOnce)
	}
}

func (m *TableStatManger) printOnce() {
	defer m.scheduleOnce()

	m.Lock()
	defer m.Unlock()

	buf := bytes.NewBuffer(nil)
	var deleteKeys []string
	// 顺便更新一下
	m.estimatedNow = util.CurrentMS()
	expireTime := m.estimatedNow - 180_000
	hasReporter := len(m.reporters) > 0
	tableMetrics := map[string]map[string]int64{}
	for table, te := range m.x {
		if te.dirty {
			te.dirty = false
			buf.Reset()
			buf.WriteString("table=")
			buf.WriteString(table)

			var copyX map[string]int64
			if hasReporter {
				copyX = make(map[string]int64, len(te.x))
			}
			for key, value := range te.x {
				if hasReporter {
					copyX[key] = value
				}
				if value > 0 {
					buf.WriteString(" ")
					buf.WriteString(key)
					buf.WriteString("=")
					buf.WriteString(strconv.FormatInt(value, 10))
					te.x[key] = 0
				}
			}
			logger.Stat(buf.String())
			if hasReporter {
				tableMetrics[table] = copyX
			}

		} else {
			// 太旧就删掉, 将key累加到deleteKeys 防止边遍历变删除
			if te.lastTouchTimestamp < expireTime {
				deleteKeys = append(deleteKeys, table)
			}
		}
	}
	for _, table := range deleteKeys {
		delete(m.x, table)
	}

	if hasReporter {
		// reporter就异步吧
		go func() {
			reportResult := ReportResult{TableMetrics: tableMetrics}
			for _, r := range m.reporters {
				r.Report(reportResult)
			}
		}()
	}
}

func (m *TableStatManger) Start() {
	m.scheduleOnce()
	go m.consumeLoop()
}

func (m *TableStatManger) consumeLoop() {
	for {

		addItem := <-m.dataChan
		// 返回说明有数据了

		// 上锁
		m.Lock()

		// 处理完第一个
		m.add0(addItem)

		// 并且drain掉整个chan
		size := len(m.dataChan)
		for i := 0; i < size; i++ {
			m.add0(<-m.dataChan)
		}

		m.Unlock()
	}
}

func (m *TableStatManger) add0(item *AddItem) {
	te, ok := m.x[item.table]
	if !ok {
		te = &TableEntry{
			x:     map[string]int64{},
			dirty: false,
		}
		m.x[item.table] = te
	}

	if item.sum {
		te.x[item.key] += item.value
	} else {
		// max
		if old, ok := te.x[item.key]; !ok || old < item.value {
			te.x[item.key] = item.value
		}
	}

	te.dirty = true
	te.lastTouchTimestamp = m.estimatedNow
}

// 虽然这个接口具备合并能力, 但可以的话还是可以在上层减少调用量
func (m *TableStatManger) Add(table string, key string, value int64) {
	// 我们这个场景不可能出现<=0的, <=0就没必要统计了
	if value <= 0 {
		return
	}

	m.dataChan <- &AddItem{
		table: table,
		key:   key,
		value: value,
		sum:   true,
	}
}

func (m *TableStatManger) Max(table string, key string, value int64) {
	// 我们这个场景不可能出现<=0的, <=0就没必要统计了
	if value <= 0 {
		return
	}

	m.dataChan <- &AddItem{
		table: table,
		key:   key,
		value: value,
		sum:   false,
	}
}

func Add(table string, key string, value int64) {
	DefaultManager.Add(table, key, value)
}
