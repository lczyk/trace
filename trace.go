package trace

import (
	"fmt"
	"runtime"
	"strings"
)

type walkable interface {
	// WalkEnter(fn enter_func) error
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

func NewTracer() Tracer {
	chain := make([]Node, 1, 32)
	chain[0] = &Enter{
		name: string(START_NODE),
	}
	return &tracer{
		stack:           chain,
		where:           chain[0].(*Enter),
		messagesEnabled: true,
	}

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

type Node interface {
	linked
}

type Enter struct {
	name string
	// next/prev nodes in the chain
	next Node
	prev Node
	// parent node -- where are we are entering from
	parent *Enter
}

func (n Enter) String() string {
	var out strings.Builder
	if n.parent != nil {
		out.WriteString(string(n.parent.name))
		out.WriteString(" -> ")
	}
	out.WriteString(string(n.name))
	return out.String()
}

func (n *Enter) Next() Node   { return n.next }
func (n *Enter) Prev() Node   { return n.prev }
func (n *Enter) Name() string { return n.name }

var (
	_ Node   = (*Enter)(nil)
	_ linked = (*Enter)(nil)
	_ namer  = (*Enter)(nil)
)

type Exit struct {
	name string
	// next/prev nodes in the chain
	next Node
	prev Node
	// parent node -- what is this an exit from
	parent *Enter
}

func (n *Exit) String() string {
	var out strings.Builder
	if n.parent != nil && n.parent.parent != nil {
		out.WriteString(string(n.parent.parent.name))
		out.WriteString(" <- ")
		out.WriteString(string(n.name))
	} else {
		out.WriteString(string(n.name))
	}
	return out.String()
}
func (n *Exit) Next() Node { return n.next }
func (n *Exit) Prev() Node { return n.prev }

func (n *Exit) Name() string { return n.name }

var (
	_ Node   = (*Exit)(nil)
	_ linked = (*Exit)(nil)
	_ namer  = (*Exit)(nil)
)

type Message struct {
	Message string
	// next/prev nodes in the chain
	next Node
	prev Node
	// parent node -- the enter node at which we are messaging
	parent *Enter
}

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
		if n.name == START_NODE {
			break
		}
		stack = append(stack, n.name)
	}
	return stack
}

func (m *Message) Next() Node { return m.next }
func (m *Message) Prev() Node { return m.prev }

// ParentName returns the name of the enclosing function scope at the time
// the message was emitted. empty if at the root.
func (m *Message) ParentName() string {
	if m.parent == nil || m.parent.name == START_NODE {
		return ""
	}
	return m.parent.name
}

var (
	_ Node   = (*Message)(nil)
	_ linked = (*Message)(nil)
)

type tracer struct {
	stack []Node
	// pointer to the enter node of the current function
	where *Enter
	// gate for Message/Messagef. enter/exit unaffected.
	messagesEnabled bool
	// set once Done has appended the closing END_NODE exit
	done bool
	// chunked arenas: pointers handed out remain valid across grows
	enters arena[Enter]
	exits  arena[Exit]
	msgs   arena[Message]
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

// Append any number of nodes to the chain
func (t *tracer) append(node ...Node) {
	for _, n := range node {
		// link new node with the top of the stack
		top := t.stack[len(t.stack)-1]
		top.(settable_next).SetNext(n)
		n.(settable_prev).SetPrev(top)
		// if we've added an new enter node, set the parent
		switch n := n.(type) {
		case *Enter:
			n.parent = t.where
			t.where = n // and update the where pointer
		case *Exit:
			t.where = n.parent.parent
		case *Message:
			// nothing to do for message nodes
		default:
			panic("unknown node type")
		}
		// actually set the next node
		t.stack = append(t.stack, n)
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
	n := t.enters.new(Enter{name: where_str})
	t.append(n)
	return t.exits.new(Exit{
		name:   where_str,
		parent: n,
	})
}

// Usage pattern: defer t.Un(t.Trace(p, "..."))
func (t *tracer) Un(exit *Exit) {
	debug("< exiting", exit.name)
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
	debug("  messaging", t.where.name)
	t.append(t.msgs.new(Message{
		Message: argsToMessage(args...),
		parent:  t.where,
	}))
}

func (t *tracer) Messagef(format string, args ...any) {
	if !t.messagesEnabled {
		return
	}
	debug("  messaging", t.where.name)
	t.append(t.msgs.new(Message{
		Message: fmt.Sprintf(format, args...),
		parent:  t.where,
	}))
}

type settable_next interface{ SetNext(Node) }
type settable_prev interface{ SetPrev(Node) }

func (t *Enter) SetNext(n Node)   { t.next = n }
func (t *Enter) SetPrev(n Node)   { t.prev = n }
func (t *Exit) SetNext(n Node)    { t.next = n }
func (t *Exit) SetPrev(n Node)    { t.prev = n }
func (t *Message) SetNext(n Node) { t.next = n }
func (t *Message) SetPrev(n Node) { t.prev = n }

var (
	_ settable_next = (*Enter)(nil)
	_ settable_next = (*Exit)(nil)
	_ settable_prev = (*Enter)(nil)
	_ settable_prev = (*Exit)(nil)
	_ settable_next = (*Message)(nil)
	_ settable_prev = (*Message)(nil)
)

func (t *tracer) Done() {
	if t.done {
		return
	}
	t.append(t.exits.new(Exit{
		name:   END_NODE,
		parent: t.stack[0].(*Enter), // link to the START_NODE
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
	// stack := make([]*Enter, 0)
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
