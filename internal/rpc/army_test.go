package rpc

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/messages"
)

func TestMoveArmyRequiresDestination(t *testing.T) {
	handler := &armyHandler{}
	_, err := handler.MoveArmy(context.Background(), connect.NewRequest(&servicev1.MoveArmyRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want %v", connect.CodeOf(err), connect.CodeInvalidArgument)
	}
}

func TestTrainingErrorRejectsInvalidTroopType(t *testing.T) {
	err := trainingError(&messages.InvalidTroopTypeError{})
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want %v", connect.CodeOf(err), connect.CodeInvalidArgument)
	}
}
