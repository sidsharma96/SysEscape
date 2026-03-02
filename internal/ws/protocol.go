package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	ProtocolVersion1 = 1

	TypeHello          = "hello"
	TypeHelloAck       = "hello_ack"
	TypeSnapshot       = "snapshot"
	TypeDelta          = "delta"
	TypeApplyAction    = "apply_action"
	TypeActionAccepted = "action_accepted"
	TypeWinUpdate      = "win_update"
	TypePing           = "ping"
	TypePong           = "pong"
)

var ErrInvalidEnvelope = errors.New("invalid ws envelope")

type Envelope struct {
	ProtocolVersion int             `json:"protocolVersion,omitempty"`
	Type            string          `json:"type"`
	RunID           string          `json:"runId,omitempty"`
	Seq             *int            `json:"seq,omitempty"`
	Payload         json.RawMessage `json:"payload,omitempty"`
}

type HelloPayload struct {
	RunToken      string `json:"runToken"`
	ResumeFromSeq *int   `json:"resumeFromSeq,omitempty"`
}

type HelloAckPayload struct {
	SnapshotRequired bool `json:"snapshotRequired"`
}

type ApplyActionPayload struct {
	ActionKey       string `json:"actionKey"`
	ClientRequestID string `json:"clientRequestId"`
	ExpectedSeq     int    `json:"expectedSeq"`
}

type ActionAcceptedPayload struct {
	ActionKey string `json:"actionKey"`
	Seq       int    `json:"seq"`
}

func DecodeEnvelope(data []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrInvalidEnvelope, err)
	}
	if strings.TrimSpace(env.Type) == "" {
		return Envelope{}, fmt.Errorf("%w: missing type", ErrInvalidEnvelope)
	}
	return env, nil
}

func ValidateClientEnvelope(env Envelope) error {
	switch env.Type {
	case TypePong:
		return nil
	case TypeHello:
		_, err := ValidateAndDecodeHello(env)
		return err
	case TypeApplyAction:
		_, err := ValidateAndDecodeApplyAction(env)
		return err
	default:
		return fmt.Errorf("%w: unsupported client message type %q", ErrInvalidEnvelope, env.Type)
	}
}

func ValidateAndDecodeHello(env Envelope) (HelloPayload, error) {
	if env.ProtocolVersion != ProtocolVersion1 {
		return HelloPayload{}, fmt.Errorf("%w: unsupported protocol version", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(env.RunID) == "" {
		return HelloPayload{}, fmt.Errorf("%w: missing runId", ErrInvalidEnvelope)
	}
	payload, err := DecodeHelloPayload(env.Payload)
	if err != nil {
		return HelloPayload{}, err
	}
	if strings.TrimSpace(payload.RunToken) == "" {
		return HelloPayload{}, fmt.Errorf("%w: missing runToken", ErrInvalidEnvelope)
	}
	return payload, nil
}

func ValidateAndDecodeApplyAction(env Envelope) (ApplyActionPayload, error) {
	if env.ProtocolVersion != ProtocolVersion1 {
		return ApplyActionPayload{}, fmt.Errorf("%w: unsupported protocol version", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(env.RunID) == "" {
		return ApplyActionPayload{}, fmt.Errorf("%w: missing runId", ErrInvalidEnvelope)
	}
	payload, err := DecodeApplyActionPayload(env.Payload)
	if err != nil {
		return ApplyActionPayload{}, err
	}
	if strings.TrimSpace(payload.ActionKey) == "" {
		return ApplyActionPayload{}, fmt.Errorf("%w: missing actionKey", ErrInvalidEnvelope)
	}
	clientRequestID, err := uuid.Parse(payload.ClientRequestID)
	if err != nil || clientRequestID.Version() != 4 {
		return ApplyActionPayload{}, fmt.Errorf("%w: clientRequestId must be uuid v4", ErrInvalidEnvelope)
	}
	return payload, nil
}

func DecodeHelloPayload(raw json.RawMessage) (HelloPayload, error) {
	var p HelloPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return HelloPayload{}, fmt.Errorf("%w: invalid hello payload: %v", ErrInvalidEnvelope, err)
	}
	return p, nil
}

func DecodeApplyActionPayload(raw json.RawMessage) (ApplyActionPayload, error) {
	var p ApplyActionPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return ApplyActionPayload{}, fmt.Errorf("%w: invalid apply_action payload: %v", ErrInvalidEnvelope, err)
	}
	return p, nil
}
