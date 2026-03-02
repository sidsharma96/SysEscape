package service

func Replay(spec SimulationSpec, log []LogEntry) (*Engine, error) {
	e := NewEngine(spec)
	for i, entry := range log {
		if entry.Seq != i+1 {
			return nil, ErrInvalidReplayLog
		}
		switch entry.ActionType {
		case ActionTypeTick:
			if entry.ActionKey != nil || entry.ClientRequestID != nil {
				return nil, ErrInvalidReplayLog
			}
			if err := e.Tick(); err != nil {
				return nil, err
			}
		case ActionTypePlayer:
			if entry.ActionKey == nil || entry.ClientRequestID == nil {
				return nil, ErrInvalidReplayLog
			}
			if err := e.ApplyAction(*entry.ActionKey, entry.ClientRequestID); err != nil {
				return nil, err
			}
		default:
			return nil, ErrInvalidReplayLog
		}
	}
	return e, nil
}
