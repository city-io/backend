package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"golang.org/x/crypto/bcrypt"

	"cityio/internal/auth"
	pb "cityio/internal/gen/cityio/v1"
	"cityio/internal/mapping"
	"cityio/internal/messages"
	"cityio/internal/services"
	"cityio/internal/stream"
)

type userHandler struct {
	srv *Server
}

func (h *userHandler) Register(ctx context.Context, req *connect.Request[pb.RegisterRequest]) (*connect.Response[pb.RegisterResponse], error) {
	userID, err := services.CreateUser(ctx, h.srv.cluster, &services.CreateUserRequest{
		Email:    req.Msg.GetEmail(),
		Username: req.Msg.GetUsername(),
		Password: req.Msg.GetPassword(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	token, err := auth.Issue(h.srv.jwtSecret, auth.Claims{
		UserID:   userID,
		Username: req.Msg.GetUsername(),
		Email:    req.Msg.GetEmail(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.RegisterResponse{UserId: userID, Token: token}), nil
}

func (h *userHandler) Login(ctx context.Context, req *connect.Request[pb.LoginRequest]) (*connect.Response[pb.LoginResponse], error) {
	res, err := h.srv.cluster.RequestDBFuture(messages.GetUserByIdentifierMessage{
		Identifier: req.Msg.GetIdentifier(),
	}).Result()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	lookup, ok := res.(messages.GetUserByIdentifierResponseMessage)
	if !ok || !lookup.Found {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid credentials"))
	}

	if err := bcrypt.CompareHashAndPassword([]byte(lookup.User.Password), []byte(req.Msg.GetPassword())); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid credentials"))
	}

	user := lookup.User
	if live, err := h.srv.cluster.Request("user", user.UserID, messages.GetUserMessage{}); err == nil {
		if resp, ok := live.(*messages.GetUserResponseMessage); ok {
			user = resp.User
		}
	}

	token, err := auth.Issue(h.srv.jwtSecret, auth.Claims{
		UserID:   user.UserID,
		Username: user.Username,
		Email:    user.Email,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.LoginResponse{
		Token: token,
		User:  mapping.UserToProto(user),
	}), nil
}

func (h *userHandler) GetUser(ctx context.Context, req *connect.Request[pb.GetUserRequest]) (*connect.Response[pb.GetUserResponse], error) {
	res, err := h.srv.cluster.Request("user", req.Msg.GetUserId(), messages.GetUserMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, ok := res.(*messages.GetUserResponseMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}
	return connect.NewResponse(&pb.GetUserResponse{User: mapping.UserToProto(resp.User)}), nil
}

func (h *userHandler) DeleteUser(ctx context.Context, req *connect.Request[pb.DeleteUserRequest]) (*connect.Response[pb.DeleteUserResponse], error) {
	if err := h.srv.cluster.Tell("user", req.Msg.GetUserId(), messages.DeleteUserMessage{UserID: req.Msg.GetUserId()}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.DeleteUserResponse{}), nil
}

func (h *userHandler) StreamState(ctx context.Context, req *connect.Request[pb.StreamStateRequest], out *connect.ServerStream[pb.UserState]) error {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("missing claims"))
	}

	ch, unsubscribe := stream.Subscribe(claims.UserID)
	defer unsubscribe()

	if res, err := h.srv.cluster.Request("user", claims.UserID, messages.GetUserMessage{}); err == nil {
		if resp, ok := res.(*messages.GetUserResponseMessage); ok {
			if err := out.Send(&pb.UserState{Gold: resp.User.Gold, Food: resp.User.Food}); err != nil {
				return err
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case state, ok := <-ch:
			if !ok {
				return nil
			}
			if err := out.Send(&pb.UserState{Gold: state.Gold, Food: state.Food}); err != nil {
				return err
			}
		}
	}
}
