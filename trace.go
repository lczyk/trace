package trace

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

type walkable interface {
	Walk(fn func(node Node) error) error
}

type namer interface {
	Name() string
}

type Tracer interface {
	// add the function to the tracing stack
	Trace(where ...string) *Exit
	Un(*Exit)
	// add a message to the trace
	Message(args ...any)
	// add a printf-style message to the trace
	Messagef(format string, args ...any)
	// get all the messages
	Messages() []Message
	// mark the end of the trace (idempotent)
	Done()
	// return the tracer as a walkable object. calls Done implicitly.
	ToWalkable() (walkable, error)
	// length of the trace stack
	Len() int
	// enable/disable message recording. enter/exit nodes are unaffected.
	SetMessagesEnabled(bool)
	// query message recording state
	MessagesEnabled() bool
}

// NewTracer constructs a new [Tracer] with message recording enabled.
// The returned tracer is not safe for concurrent use; wrap with
// [NewSyncTracer] if multiple goroutines need to share one.
func NewTracer() Tracer {
	t := &tracer{messagesEnabled: true}
	rootIdx := t.intern(START_NODE)
	root := &Enter{nameIdx: rootIdx, owner: t, idx: 0}
	t.stack = make([]Node, 1, 32)
	t.stack[0] = root
	t.where = root
	return t
}

// NewTracerWithTime constructs a [Tracer] that stamps every Enter,
// Exit, and Message node with [time.Time] at the moment of recording.
// Adds one [time.Since] call per recorded node compared to [NewTracer].
// Required for downstream consumers that want per-scope durations.
func NewTracerWithTime() Tracer {
	t := NewTracer().(*tracer)
	t.timed = true
	t.start = time.Now()
	// parallel timestamps slice. tns[0] = 0 for the START node.
	t.tns = make([]int64, 1, 32)
	return t
}

// local debug flag

const dEBUG = false

// const dEBUG = true

func debug(args ...any) {
	if dEBUG {
		fmt.Println(args...)
	}
}

//////////

const (
	START_NODE string = "<START>"
	END_NODE   string = "<END>"
)

type linked interface {
	Next() Node
	Prev() Node
}

// Node is the common type for every entry in a recorded trace:
// [Enter], [Exit], and [Message]. Use a type switch in a [Tracer.Walk]
// callback to dispatch on concrete kind.
type Node interface {
	linked
}

// Enter is a node marking the beginning of a traced scope.
type Enter struct {
	nameIdx uint32
	idx     uint32
	owner   *tracer
	parent  *Enter
}

// Time returns the wall-clock entry time, or zero if the tracer was not
// constructed with [NewTracerWithTime].
func (n *Enter) Time() time.Time { return nodeTime(n.owner, n.idx) }

func (n Enter) String() string {
	var out strings.Builder
	if n.parent != nil {
		out.WriteString(n.parent.Name())
		out.WriteString(" -> ")
	}
	out.WriteString(n.Name())
	return out.String()
}

func (n *Enter) Next() Node   { return nodeNext(n.owner, n.idx) }
func (n *Enter) Prev() Node   { return nodePrev(n.owner, n.idx) }
func (n *Enter) Name() string { return nodeName(n.owner, n.nameIdx) }

var (
	_ Node   = (*Enter)(nil)
	_ linked = (*Enter)(nil)
	_ namer  = (*Enter)(nil)
)

// Exit is a node marking the end of a traced scope, paired with the
// [Enter] returned by [Tracer.Trace] via [Tracer.Un].
type Exit struct {
	nameIdx uint32
	idx     uint32
	owner   *tracer
	parent  *Enter
}

// Time returns the wall-clock exit time, or zero if the tracer was not
// constructed with [NewTracerWithTime].
func (n *Exit) Time() time.Time { return nodeTime(n.owner, n.idx) }

func (n *Exit) String() string {
	var out strings.Builder
	if n.parent != nil && n.parent.parent != nil {
		out.WriteString(n.parent.parent.Name())
		out.WriteString(" <- ")
		out.WriteString(n.Name())
	} else {
		out.WriteString(n.Name())
	}
	return out.String()
}

func (n *Exit) Next() Node   { return nodeNext(n.owner, n.idx) }
func (n *Exit) Prev() Node   { return nodePrev(n.owner, n.idx) }
func (n *Exit) Name() string { return nodeName(n.owner, n.nameIdx) }

var (
	_ Node   = (*Exit)(nil)
	_ linked = (*Exit)(nil)
	_ namer  = (*Exit)(nil)
)

// Message is an inline note attached to the current scope at the time
// it was recorded.
type Message struct {
	Message string
	idx     uint32
	owner   *tracer
	parent  *Enter
}

// Time returns the wall-clock message time, or zero if the tracer was
// not constructed with [NewTracerWithTime].
func (m *Message) Time() time.Time { return nodeTime(m.owner, m.idx) }

func (m Message) String() string {
	var out strings.Builder
	stack := m.Stack()
	for j := len(stack) - 1; j >= 0; j-- {
		out.WriteString(fmt.Sprintf("%s:", stack[j]))
	}
	out.WriteString(fmt.Sprintf(" %s", m.Message))
	return out.String()
}

func (m *Message) Stack() []string {
	stack := make([]string, 0)
	for n := m.parent; n != nil; n = n.parent {
		if n.Name() == START_NODE {
			break
		}
		stack = append(stack, n.Name())
	}
	return stack
}

func (m *Message) Next() Node { return nodeNext(m.owner, m.idx) }
func (m *Message) Prev() Node { return nodePrev(m.owner, m.idx) }

// ParentName returns the name of the enclosing function scope at the time
// the message was emitted. empty if at the root.
func (m *Message) ParentName() string {
	if m.parent == nil {
		return ""
	}
	name := m.parent.Name()
	if name == START_NODE {
		return ""
	}
	return name
}

var (
	_ Node   = (*Message)(nil)
	_ linked = (*Message)(nil)
)

// nodeNext / nodePrev / nodeTime / nodeName are shared helpers backing
// the per-node methods. The neighbour lookups go through the owner
// tracer's stack so the linked-list pointers don't have to live on the
// node structs themselves -- saving 32B/node (next + prev iface words).
func nodeNext(t *tracer, idx uint32) Node {
	if t == nil {
		return nil
	}
	next := int(idx) + 1
	if next >= len(t.stack) {
		return nil
	}
	return t.stack[next]
}

func nodePrev(t *tracer, idx uint32) Node {
	if t == nil || idx == 0 {
		return nil
	}
	return t.stack[idx-1]
}

func nodeTime(t *tracer, idx uint32) time.Time {
	if t == nil || !t.timed {
		return time.Time{}
	}
	return t.start.Add(time.Duration(t.tns[idx]))
}

func nodeName(t *tracer, nameIdx uint32) string {
	if t == nil {
		return ""
	}
	return t.names[nameIdx]
}

type tracer struct {
	stack []Node
	// pointer to the enter node of the current function
	where *Enter
	// gate for Message/Messagef. enter/exit unaffected.
	messagesEnabled bool
	// set once Done has appended the closing END_NODE exit
	done bool
	// when true, every appended node gets a timestamp pushed into tns
	timed bool
	// per-tracer string pool for Enter/Exit names (interned). Indexed by
	// nameIdx fields on Enter/Exit. Saves bytes when the same scope name
	// is recorded many times (typical for parsers / recursive descent).
	names   []string
	nameMap map[string]uint32
	// single-entry intern cache; hits when the same name is interned twice
	// in a row (typical for recursive descent parsers).
	lastInternStr string
	lastInternIdx uint32
	// parallel slice of nanosecond offsets from start, indexed by node.idx.
	// nil unless timed.
	tns   []int64
	start time.Time
	// chunked arenas: pointers handed out remain valid across grows
	enters arena[Enter]
	exits  arena[Exit]
	msgs   arena[Message]
}

// intern looks up s in the per-tracer name pool, returning its index.
// A single-entry cache (lastInternStr / lastInternIdx) short-circuits
// the map lookup when the same name is interned consecutively -- the
// common case for parsers and other recursive-descent workloads that
// re-enter the same scope thousands of times.
func (t *tracer) intern(s string) uint32 {
	if s == t.lastInternStr && t.lastInternStr != "" {
		return t.lastInternIdx
	}
	var i uint32
	if existing, ok := t.nameMap[s]; ok {
		i = existing
	} else {
		if t.nameMap == nil {
			t.nameMap = make(map[string]uint32, 32)
		}
		i = uint32(len(t.names))
		t.names = append(t.names, s)
		t.nameMap[s] = i
	}
	t.lastInternStr = s
	t.lastInternIdx = i
	return i
}

// arenaChunk is the per-chunk capacity. tuned for typical trace depths.
const arenaChunk = 64

// arena is a chunked bump allocator. *T pointers handed out remain valid
// as new chunks are appended (unlike a single growable slice, where a
// realloc would invalidate prior pointers).
type arena[T any] struct {
	chunks [][]T
	idx    int
}

func (a *arena[T]) new(v T) *T {
	if len(a.chunks) == 0 || a.idx == arenaChunk {
		a.chunks = append(a.chunks, make([]T, arenaChunk))
		a.idx = 0
	}
	c := a.chunks[len(a.chunks)-1]
	c[a.idx] = v
	p := &c[a.idx]
	a.idx++
	return p
}

func (t *tracer) SetMessagesEnabled(b bool) { t.messagesEnabled = b }
func (t *tracer) MessagesEnabled() bool     { return t.messagesEnabled }

// append assigns the idx + owner backref, links parent for Enter/Exit,
// pushes the node onto the stack, and stamps a timestamp into tns if
// the tracer is timed.
func (t *tracer) append(node ...Node) {
	for _, n := range node {
		idx := uint32(len(t.stack))
		switch n := n.(type) {
		case *Enter:
			n.idx = idx
			n.owner = t
			n.parent = t.where
			t.where = n
		case *Exit:
			n.idx = idx
			n.owner = t
			t.where = n.parent.parent
		case *Message:
			n.idx = idx
			n.owner = t
		default:
			panic("unknown node type")
		}
		t.stack = append(t.stack, n)
		if t.timed {
			t.tns = append(t.tns, time.Since(t.start).Nanoseconds())
		}
	}
}

func callerName(N int) string {
	parent, _, _, _ := runtime.Caller(N + 1)
	info := runtime.FuncForPC(parent)
	name := info.Name()
	return name
}

func here(name string) string {
	// strip everything before the last . to get just the function name
	name = name[strings.LastIndex(name, ".")+1:]
	return name
}

// Here returns the unqualified name of the calling function. Pair with
// [Tracer.Trace] when you want a stable scope name without spelling it
// out.
//
//go:noinline
func Here() string {
	return here(callerName(1))
}

//go:inline
func whereToString(where_args ...string) string {
	var where string
	if len(where_args) == 0 {
		where = here(callerName(2))
	} else if len(where_args) == 1 {
		where = where_args[0]
	} else { // len(where_args) > 1
		format := where_args[0]
		rest := make([]any, len(where_args)-1)
		for i, v := range where_args[1:] {
			rest[i] = v
		}
		where = fmt.Sprintf(format, rest...)
	}
	return where
}

func (t *tracer) Trace(where ...string) *Exit {
	where_str := whereToString(where...)
	debug("> entering", where_str)
	nameIdx := t.intern(where_str)
	n := t.enters.new(Enter{nameIdx: nameIdx})
	t.append(n)
	return t.exits.new(Exit{nameIdx: nameIdx, parent: n})
}

// Usage pattern: defer t.Un(t.Trace(p, "..."))
func (t *tracer) Un(exit *Exit) {
	debug("< exiting")
	t.append(exit)
}

func argsToMessage(args ...any) string {
	// check if the first
	var msg string
	if len(args) == 0 {
		msg = "<empty message>"
	} else if len(args) == 1 {
		// check if we are a function func() string
		if fn, ok := args[0].(func() string); ok {
			msg = fn()
		} else if str, ok := args[0].(string); ok {
			msg = str
		} else {
			msg = fmt.Sprintf("%v", args[0])
		}
	} else {
		msg = fmt.Sprint(args...)
	}
	return msg
}

func (t *tracer) Message(args ...any) {
	if !t.messagesEnabled {
		return
	}
	debug("  messaging")
	t.append(t.msgs.new(Message{
		Message: argsToMessage(args...),
		parent:  t.where,
	}))
}

func (t *tracer) Messagef(format string, args ...any) {
	if !t.messagesEnabled {
		return
	}
	debug("  messaging")
	t.append(t.msgs.new(Message{
		Message: fmt.Sprintf(format, args...),
		parent:  t.where,
	}))
}

func (t *tracer) Done() {
	if t.done {
		return
	}
	nameIdx := t.intern(END_NODE)
	t.append(t.exits.new(Exit{
		nameIdx: nameIdx,
		parent:  t.stack[0].(*Enter), // link to the START_NODE
	}))
	t.done = true
}

func (t *tracer) ToWalkable() (walkable, error) {
	if t.stack == nil {
		panic("call stack is empty")
	}
	t.Done()
	return t, nil
}

func (t *tracer) Messages() []Message {
	messages := make([]Message, 0)
	walkable, err := t.ToWalkable()
	if err != nil {
		return nil
	}

	_ = walkable.Walk(func(n Node) error {
		switch n := n.(type) {
		case *Message:
			messages = append(messages, *n)
		default:
			// do nothing
		}
		return nil
	})
	return messages
}

func (t *tracer) Walk(fn func(node Node) error) error {
	var err error
	for i := 0; i < len(t.stack); i++ {
		node := t.stack[i]
		err = fn(node)
		if err != nil {
			return fmt.Errorf("error in walk function: %w", err)
		}
	}
	return nil
}

func (t *tracer) Len() int {
	if t.stack == nil {
		return 0
	}
	return len(t.stack)
}

var (
	_ Tracer   = (*tracer)(nil)
	_ walkable = (*tracer)(nil)
)
