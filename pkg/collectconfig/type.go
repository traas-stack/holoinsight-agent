package collectconfig

const (
	EElectLine      = "line"
	EElectRefIndex  = "refIndex"
	EElectRefName   = "refName"
	EElectLeftRight = "leftRight"
	EElectRegexp    = "regexp"
	EElectRefMeta   = "refMeta"
	EElectPathVar   = "pathvar"
)

type (
	// 1. pre-where: 可选, 只能支持左起右至的elect
	// 2. structure: 可选, 使得数据结构化
	// 3. 常规处理: select [...] from {} where {} group by [...] having {} window {} output {}
	// 日志结构化
	LogStructure struct {
		Type      string
		Separator string
	}

	// 定义变量
	Vars struct {
		Vars []Var
	}
	Var struct {
		Name  string
		Elect *Elect
		// 这里支持流水线
		Pipeline *Pipeline
	}
	// TODO 改个名字 容易误解
	Pipeline struct {
		// 对值做转换
		Transform []*Transform
		// 当转换结果是nil或empty时, 使用使用该默认值代替
		DefaultValue string
		// 执行出错是否丢弃记录
		DiscardRecordWhenError bool
		// 如果执行完pipeline的结果是null
		DiscardRecordWhenResultIsNil bool
	}
	// 这里定义很多转换, 可以对原有的值进行转换
	// 入参一般是 string 或 float64, 或其他(需要明确指出)
	Transform struct {
		Type string
		Arg  string
	}
	Select struct {
		Values []*SelectOne `json:"values"`
	}
	SelectOne struct {
		// TODO
		DataType string `json:"dataType"`

		As string `json:"as"`
		// count
		// elect
		Type  string `json:"type"`
		Elect *Elect `json:"elect"`
		// agg
		Agg   string `json:"agg"`
		Where *Where `json:"where"`
	}
	From struct {
		Type        string           `json:"type"`
		Log         *FromLog         `json:"log"`
		ProcessPerf *FromProcessPerf `json:"processPerf"`
	}
	FromProcessPerf struct {
		IncludeUsernames []string
		ExcludeUsernames []string
		IncludeProcesses []string
		ExcludeProcesses []string
		IncludeKeywords  []string
		ExcludeKeywords  []string
	}
	GroupBy struct {
		Groups      []*Group         `json:"groups"`
		MaxKeySize  int              `json:"maxKeySize"`
		LogAnalysis *LogAnalysisConf `json:"logAnalysis"`
	}
	Window struct {
		// 5s 5000
		Interval interface{} `json:"interval"`
	}
	Output struct {
		Type    string   `json:"type"`
		Gateway *Gateway `json:"gateway"`
	}
	Gateway struct {
		// 用户可以覆盖, 否则默认使用 tableName
		MetricName string `json:"metricName"`
	}
	FromLog struct {
		Path []*FromLogPath `json:"path"`
		// defaults to UTF8
		Charset string        `json:"charset"`
		Parse   *FromLogParse `json:"parse"`
		// 定义时间戳如何解析
		Time      *TimeConf         `json:"time"`
		Multiline *FromLogMultiline `json:"multiline"`
	}
	// TODO 用于支持多文件
	FromLogPaths struct {
		// 99% 的情况 paths里应该有且只有一项 type==simple 的绝对路径
		// 另外所有被匹配的文件都是按照相同的规则执行的, 因此这些文件应该具有某种共性, 比如相同格式
		Paths []*FromLogPath `json:"paths"`
		// 多久监听一下文件tree变化, 对于 glob/regexp 有效
		WatchInterval string `json:"watchInterval"`
	}
	FromLogPath struct {
		// path/glob/regexp/'glob&regexp'
		// 当type==regexp 时, 需要制定一个dir, 该dir的所有子孙文件满足regexp的都会纳入采集范围
		// 这个方式比较容易出问题:
		// 1. 递归遍历代价大/符号死循环/符号导致重复采集
		// 当type=='glob&regexp' 时表示先执行glob, 再执行regexp, 这样可以使用glob的简单方式圈定一些文件
		// 然后再使用regexp更精细的匹配文件
		Type string `json:"type"`
		// used when type==path
		// used when type==format
		// /home/admin/logs/foo/{time:yyyy}/{time:MM}/{time:dd}/{time:HH}/foo.log
		// /home/admin/logs/foo/{time:yy}/{time:MM}/{time:dd}/{time:HH}/foo.log
		// used when type==glob
		// used when type==regexp
		Pattern string `json:"pattern"`
		// used when type==regexp
		Dir string `json:"dir"`
		// Limit how many files can be matched using this FromLogPath object.
		// 0 means agent defaults (maybe 10)
		MaxMatched int `json:"maxMatched"`
	}
	// TODO 遵循 logstash 风格
	FromLogMultiline struct {
		// 多行日志是否启动
		Enabled bool `json:"enabled"`
		// 行首的判断条件, 满足这个条件的就是行首
		// 比如一种方式是断言行首配 ^yyyy-MM-dd 的格式
		Where *Where `json:"where"`
		// limit max logs in a log group
		MaxLines int    `json:"maxLines"`
		What     string `json:"what"`
	}
	FromLogParse struct {
		// 有的parse代价太大, 可以在parse前做一次过滤减少parse的量
		// 此时where里仅能使用 leftRight 类型的切分
		Where *Where `json:"where,omitempty"`
		// free/separator/regexp/json
		Type string `json:"type,omitempty"`
		// 基于分隔符
		Separator *LogParseSeparator `json:"separator,omitempty"`
		Regexp    *LogParseRegexp    `json:"regexp,omitempty"`
		Grok      *LogParseGrok      `json:"grok,omitempty"`
	}
	TimeConf struct {
		// auto/processTime/elect
		Type  string `json:"type"`
		Elect *Elect `json:"elect"`
		// unix/unixMilli/golangLayout
		Format string `json:"format"`
		Layout string `json:"layout"`
		// timezone
		Timezone string `json:"timezone"`
	}
	LogParseRegexp struct {
		// 正则表达式
		Expression string `json:"expression"`
	}
	LogParseGrok struct {
		// grok表达式
		Expression string `json:"expression"`
	}
	LogParseSeparator struct {
		// 简单分隔符
		Separator string `json:"separator"`
	}

	// Elect 表示如何从当前的数据(可能是个日志行或结构化数据)里提取出想要的字段
	// Elect出的结果默认是string
	// 有少数情况结果是float64或其他, 这个需要每个type明确说明
	Elect struct {
		// refIndex/refName: 引用一个已有的字段
		// leftRight: 使用左起右至切出一个字段
		// regexp
		Type      string       `json:"type,omitempty"`
		Line      *ElectLine   `json:"line,omitempty"`
		RefIndex  *RefIndex    `json:"refIndex,omitempty"`
		RefName   *RefName     `json:"refName,omitempty"`
		LeftRight *LeftRight   `json:"leftRight,omitempty"`
		Regexp    *ElectRegexp `json:"regexp,omitempty"`
		// TODO 切出来字段后可以做流水线 待实现
		Pipeline *Pipeline     `json:"pipeline,omitempty"`
		RefMeta  *ElectRegMeta `json:"refMeta,omitempty"`
		PathVar  *ElectPathVar `json:"pathVar,omitempty"`
	}
	ElectPathVar struct {
		Name string `json:"name"`
	}
	ElectRegMeta struct {
		Name string `json:"name"`
	}
	ElectLine struct {
		Index int `json:"index"`
	}
	// 考虑到性能, 除非正则表达式逻辑比较简单, 否则一般不太推荐用这个方式
	ElectRegexp struct {
		// 正则表达式
		Expression string `json:"expression"`
		// 捕获组索引
		Index int `json:"index"`
		// 捕获组名, 非空情况下优先级比 index 高
		Name string `json:"name"`
	}
	Group struct {
		// 组名
		Name string `json:"name"`
		// 如何选取组的值
		// 默认情况下 如果发生错误
		Elect *Elect `json:"elect"`
	}
	RefIndex struct {
		Index int `json:"index"`
	}
	RefName struct {
		Name string `json:"name"`
	}
	LeftRight struct {
		LeftIndex             int    `json:"leftIndex,omitempty"`
		Left                  string `json:"left,omitempty"`
		Right                 string `json:"right,omitempty"`
		MatchToEndIfMissRight bool   `json:"matchToEndIfMissRight,omitempty"`
		// 如果左起右至切不到结果那么用这个default
		// 如果找不到左起 则返回这个
		DefaultValue *string `json:"defaultValue,omitempty"`
	}
	Where struct {
		And           []*Where        `json:"and,omitempty"`
		Or            []*Where        `json:"or,omitempty"`
		Not           *Where          `json:"not,omitempty"`
		Contains      *MContains      `json:"contains,omitempty"`
		ContainsAny   *MContainsAny   `json:"containsAny,omitempty"`
		In            *MIn            `json:"in,omitempty"`
		NumberBetween *MNumberBetween `json:"numberBetween,omitempty"`
		Regexp        *MRegexp        `json:"regexp,omitempty"`
		NumberOp      *MNumberOp      `json:"numberOp,omitempty"`
	}
	MNumberOp struct {
		Elect *Elect   `json:"elect"`
		Gt    *float64 `json:"gt"`
		Gte   *float64 `json:"gte"`
		Lt    *float64 `json:"lt"`
		Lte   *float64 `json:"lte"`
		//Eqi   *int64
		//Nei   *int64
	}
	MRegexp struct {
		Elect      *Elect `json:"elect"`
		Expression string `json:"expression"`
		Multiline  bool   `json:"multiline"`
	}
	MNumberBetween struct {
		Elect       *Elect  `json:"elect"`
		Min         float64 `json:"min"`
		Max         float64 `json:"max"`
		MinIncluded bool    `json:"minIncluded"`
		MaxIncluded bool    `json:"maxIncluded"`
		// 数值是否是整数
		ParseNumberToInt bool `json:"parseNumberToInt"`
	}
	MContains struct {
		Elect      *Elect `json:"elect"`
		Value      string `json:"value"`
		Multiline  bool   `json:"multiline"`
		IgnoreCase bool   `json:"ignoreCase"`
	}
	MContainsAny struct {
		Elect      *Elect   `json:"elect"`
		Values     []string `json:"values"`
		Multiline  bool     `json:"multiline"`
		IgnoreCase bool     `json:"ignoreCase"`
	}
	MIn struct {
		Elect      *Elect   `json:"elect"`
		Values     []string `json:"values"`
		IgnoreCase bool     `json:"ignoreCase"`
	}
	ExecuteRule struct {
		Type string `json:"type"`
		// 5s 5000单位毫秒
		FixedRate interface{} `json:"fixedRate"`
	}
	// SQL style task
	SQLTask struct {
		Select      *Select      `json:"select"`
		From        *From        `json:"from"`
		Where       *Where       `json:"where"`
		GroupBy     *GroupBy     `json:"groupBy"`
		Window      *Window      `json:"window"`
		Output      *Output      `json:"output"`
		ExecuteRule *ExecuteRule `json:"executeRule"`
	}
	MetricConfig struct {
		Name       string `json:"name"`
		MetricType string `json:"metricType"`
	}
)

var (
	// 瞬时值
	MetricTypeGauge = "GAUGE"
	// 增量
	MetricTypeCounter = "COUNTER"
	// 增量除以采集周期 得到的是速度 increment/s
	MetricTypeCounterByTime = "COUNTER_BY_TIME"

	CElectLine = &Elect{
		Type: EElectLine,
	}
)
