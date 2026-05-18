package trace

import "sync"

// SyncTracer wraps a [Tracer] with a mutex so multiple goroutines may
// share it. Note that the call-tree it records will interleave entries
// from concurrent callers; the tree shape only reflects program flow
// faithfully when used from a single goroutine.
type SyncTracer struct {
	mu sync.Mutex
	t  Tracer
}

// NewSyncTracer wraps t for safe concurrent use.
func NewSyncTracer(t Tracer) *SyncTracer { return &SyncTracer{t: t} }

func (s *SyncTracer) Trace(where ...string) *Exit {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.t.Trace(where...)
}

func (s *SyncTracer) Un(e *Exit) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t.Un(e)
}

func (s *SyncTracer) Message(args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t.Message(args...)
}

func (s *SyncTracer) Messagef(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t.Messagef(format, args...)
}

func (s *SyncTracer) Messages() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.t.Messages()
}

func (s *SyncTracer) Done() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t.Done()
}

func (s *SyncTracer) ToWalkable() (walkable, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.t.ToWalkable()
}

func (s *SyncTracer) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.t.Len()
}

func (s *SyncTracer) SetMessagesEnabled(b bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t.SetMessagesEnabled(b)
}

func (s *SyncTracer) MessagesEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.t.MessagesEnabled()
}

var _ Tracer = (*SyncTracer)(nil)
