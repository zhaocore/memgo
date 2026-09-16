package memory

// 事件通知端口: core 不做 HTTP, 外部投递 (webhook) 由调用方实现 EventSink 注入。
// nil sink = 无通知, 行为与未注入完全一致。

// 事件类型 (与历史库 event 列同词表; webhook 侧映射为 memory_add 等连字符形式)。
const (
	EventAdd        = "ADD"
	EventUpdate     = "UPDATE"
	EventDelete     = "DELETE"
	EventCategorize = "CATEGORIZE"
)

// Event 单条记忆事件 (payload 面对齐 custom-categories/webhooks 语义)。
type Event struct {
	Type     string // ADD | UPDATE | DELETE | CATEGORIZE
	MemoryID string
	Data     string // 记忆内容 (UPDATE=新值, DELETE=旧值; CATEGORIZE 为空)
	Category string // 命中分类 (仅 CATEGORIZE)
	EventAt  string // isoformat UTC
}

// EventSink 事件出口端口。
type EventSink interface {
	Emit(Event)
}

// emit 安全投递 (nil sink 无操作; sink 自行决定同步/异步)。
func (m *Memory) emit(ev Event) {
	if m.Sink == nil {
		return
	}
	if ev.EventAt == "" {
		ev.EventAt = nowUTC()
	}
	m.Sink.Emit(ev)
}

// emitAdd ADD 事件 (data 为记忆文本)。
func (m *Memory) emitAdd(memoryID, data string) {
	m.emit(Event{Type: EventAdd, MemoryID: memoryID, Data: data})
}

// emitUpdate UPDATE 事件 (data 为更新后文本)。
func (m *Memory) emitUpdate(memoryID, data string) {
	m.emit(Event{Type: EventUpdate, MemoryID: memoryID, Data: data})
}

// emitDelete DELETE 事件 (data 为删除前文本)。
func (m *Memory) emitDelete(memoryID, data string) {
	m.emit(Event{Type: EventDelete, MemoryID: memoryID, Data: data})
}

// emitCategorize CATEGORIZE 事件 (分类打标命中时)。
func (m *Memory) emitCategorize(memoryID, category string) {
	m.emit(Event{Type: EventCategorize, MemoryID: memoryID, Category: category})
}
