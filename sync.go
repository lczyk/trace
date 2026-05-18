package trace

import "sync"

// SyncTracer wraps a [Tracer] with a mutex so multiple goroutines may
// share it. Note that the call-tree it records will interleave entries
// from concurrent callers; the tree shape only reflects program flow
// faithfully when used from a single goroutine.
//
// Methods unlock explicitly rather than via defer to keep the
// per-method overhead minimal. The wrapped tracer's methods do not
// panic during normal use, so the missing defer cannot leak the lock.
type SyncTracer struct {
	mu sync.Mutex
	t  Tracer
}

// NewSyncTracer wraps t for safe concurrent use.
func NewSyncTracer(t Tracer) *SyncTracer { return &SyncTracer{t: t} }

func (s *SyncTracer) Trace(where ...string) *Exit {
	s.mu.Lock()
	r := s.t.Trace(where...)
	s.mu.Unlock()
	return r
}

func (s *SyncTracer) Un(e *Exit) {
	s.mu.Lock()
	s.t.Un(e)
	s.mu.Unlock()
}

func (s *SyncTracer) Message(args ...any) {
	s.mu.Lock()
	s.t.Message(args...)
	s.mu.Unlock()
}

func (s *SyncTracer) Messagef(format string, args ...any) {
	s.mu.Lock()
	s.t.Messagef(format, args...)
	s.mu.Unlock()
}

func (s *SyncTracer) Messages() []Message {
	s.mu.Lock()
	r := s.t.Messages()
	s.mu.Unlock()
	return r
}

func (s *SyncTracer) Done() {
	s.mu.Lock()
	s.t.Done()
	s.mu.Unlock()
}

func (s *SyncTracer) ToWalkable() (walkable, error) {
	s.mu.Lock()
	w, err := s.t.ToWalkable()
	s.mu.Unlock()
	return w, err
}

func (s *SyncTracer) Len() int {
	s.mu.Lock()
	r := s.t.Len()
	s.mu.Unlock()
	return r
}

func (s *SyncTracer) SetMessagesEnabled(b bool) {
	s.mu.Lock()
	s.t.SetMessagesEnabled(b)
	s.mu.Unlock()
}

func (s *SyncTracer) MessagesEnabled() bool {
	s.mu.Lock()
	r := s.t.MessagesEnabled()
	s.mu.Unlock()
	return r
}

var _ Tracer = (*SyncTracer)(nil)
