package service

func (e *Engine) ApplyAction(actionKey string, clientRequestID *string) error {
	if e.won {
		return ErrRunCompleted
	}
	if _, ok := e.spec.ActionEffects[actionKey]; !ok {
		return ErrUnknownAction
	}
	e.activeActions = append(e.activeActions, actionKey)
	e.appendLog(ActionTypePlayer, &actionKey, clientRequestID)
	return nil
}
