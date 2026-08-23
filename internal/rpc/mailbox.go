package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"cityio/internal/auth"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/mapping"
	"cityio/internal/persistence"
	"cityio/internal/stream"
)

type mailboxHandler struct {
	srv *Server
}

func (h *mailboxHandler) ListMailboxMessages(ctx context.Context, _ *connect.Request[servicev1.ListMailboxMessagesRequest]) (*connect.Response[servicev1.ListMailboxMessagesResponse], error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing claims"))
	}
	messages, err := h.srv.store.GetMailboxMessagesByRecipient(ctx, claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &servicev1.ListMailboxMessagesResponse{}
	for _, message := range messages {
		response.Messages = append(response.Messages, mapping.MailboxMessageToProto(message))
	}
	return connect.NewResponse(response), nil
}

func (h *mailboxHandler) MarkMailboxMessageRead(ctx context.Context, req *connect.Request[servicev1.MarkMailboxMessageReadRequest]) (*connect.Response[servicev1.MarkMailboxMessageReadResponse], error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing claims"))
	}
	messageID := req.Msg.GetMailboxMessageId().GetValue()
	if messageID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("mailbox message id is required"))
	}
	message, err := h.srv.store.MarkMailboxMessageRead(ctx, messageID, claims.UserID)
	if errors.Is(err, persistence.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("mailbox message not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	stream.Publish(claims.UserID, stream.StateUpdate{MailboxMessage: message})
	return connect.NewResponse(&servicev1.MarkMailboxMessageReadResponse{Message: mapping.MailboxMessageToProto(*message)}), nil
}
