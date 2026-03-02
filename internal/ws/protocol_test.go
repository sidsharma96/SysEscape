package ws

import (
	"encoding/json"
	"testing"
)

func TestDecodeEnvelope(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid", input: `{"protocolVersion":1,"type":"hello","runId":"r1","payload":{"runToken":"t"}}`},
		{name: "invalid json", input: `{`, wantErr: true},
		{name: "missing type", input: `{"protocolVersion":1}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeEnvelope([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("DecodeEnvelope() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateClientEnvelope(t *testing.T) {
	helloPayload, _ := json.Marshal(HelloPayload{RunToken: "tok"})
	actionPayload, _ := json.Marshal(ApplyActionPayload{ActionKey: "a", ClientRequestID: "6ba7b810-9dad-41d1-80b4-00c04fd430c8", ExpectedSeq: 1})

	tests := []struct {
		name string
		env  Envelope
		ok   bool
	}{
		{name: "hello ok", env: Envelope{ProtocolVersion: ProtocolVersion1, Type: TypeHello, RunID: "r", Payload: helloPayload}, ok: true},
		{name: "action ok", env: Envelope{ProtocolVersion: ProtocolVersion1, Type: TypeApplyAction, RunID: "r", Payload: actionPayload}, ok: true},
		{name: "pong ok", env: Envelope{Type: TypePong}, ok: true},
		{name: "unknown type", env: Envelope{Type: "weird"}},
		{name: "hello missing token", env: Envelope{ProtocolVersion: ProtocolVersion1, Type: TypeHello, RunID: "r", Payload: []byte(`{}`)}},
		{name: "action bad uuid", env: Envelope{ProtocolVersion: ProtocolVersion1, Type: TypeApplyAction, RunID: "r", Payload: []byte(`{"actionKey":"a","clientRequestId":"x","expectedSeq":1}`)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClientEnvelope(tt.env)
			if (err == nil) != tt.ok {
				t.Fatalf("ValidateClientEnvelope() err=%v ok=%v", err, tt.ok)
			}
		})
	}
}

func TestDecodePayloadHelpers(t *testing.T) {
	hello, err := DecodeHelloPayload([]byte(`{"runToken":"tok","resumeFromSeq":3}`))
	if err != nil {
		t.Fatalf("DecodeHelloPayload() err=%v", err)
	}
	if hello.RunToken != "tok" || hello.ResumeFromSeq == nil || *hello.ResumeFromSeq != 3 {
		t.Fatalf("DecodeHelloPayload() got=%+v", hello)
	}

	action, err := DecodeApplyActionPayload([]byte(`{"actionKey":"act","clientRequestId":"6ba7b810-9dad-41d1-80b4-00c04fd430c8","expectedSeq":9}`))
	if err != nil {
		t.Fatalf("DecodeApplyActionPayload() err=%v", err)
	}
	if action.ActionKey != "act" || action.ExpectedSeq != 9 {
		t.Fatalf("DecodeApplyActionPayload() got=%+v", action)
	}
}
