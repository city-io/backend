package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"golang.org/x/crypto/bcrypt"

	"cityio/internal/auth"
	entityv1 "cityio/internal/gen/cityio/entity/v1"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/mapping"
	"cityio/internal/messages"
	"cityio/internal/persistence"
	"cityio/internal/services"
	"cityio/internal/stream"
)

type userHandler struct {
	srv *Server
}

func (h *userHandler) Register(ctx context.Context, req *connect.Request[servicev1.RegisterRequest]) (*connect.Response[servicev1.RegisterResponse], error) {
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

	return connect.NewResponse(&servicev1.RegisterResponse{UserId: mapping.ToUserId(userID), Token: token}), nil
}

func (h *userHandler) Login(ctx context.Context, req *connect.Request[servicev1.LoginRequest]) (*connect.Response[servicev1.LoginResponse], error) {
	found, err := h.srv.store.GetUserByIdentifier(ctx, req.Msg.GetIdentifier())
	if errors.Is(err, persistence.ErrNotFound) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid credentials"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(found.Password), []byte(req.Msg.GetPassword())); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid credentials"))
	}

	user := *found
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

	return connect.NewResponse(&servicev1.LoginResponse{
		Token: token,
		User:  mapping.UserToProto(user),
	}), nil
}

func (h *userHandler) GetUser(ctx context.Context, req *connect.Request[servicev1.GetUserRequest]) (*connect.Response[servicev1.GetUserResponse], error) {
	res, err := h.srv.cluster.Request("user", req.Msg.GetUserId().GetValue(), messages.GetUserMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, ok := res.(*messages.GetUserResponseMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}
	return connect.NewResponse(&servicev1.GetUserResponse{User: mapping.UserToProto(resp.User)}), nil
}

func (h *userHandler) DeleteUser(ctx context.Context, req *connect.Request[servicev1.DeleteUserRequest]) (*connect.Response[servicev1.DeleteUserResponse], error) {
	uid := req.Msg.GetUserId().GetValue()
	if err := h.srv.cluster.Tell("user", uid, messages.DeleteUserMessage{UserID: uid}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&servicev1.DeleteUserResponse{}), nil
}

func (h *userHandler) StreamState(ctx context.Context, req *connect.Request[servicev1.StreamStateRequest], out *connect.ServerStream[servicev1.StreamStateResponse]) error {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("missing claims"))
	}

	ch, unsubscribe := stream.Subscribe(claims.UserID)
	defer unsubscribe()

	// Send initial snapshot: user, owned cities, and their buildings.
	if res, err := h.srv.cluster.Request("user", claims.UserID, messages.GetUserMessage{}); err == nil {
		if resp, ok := res.(*messages.GetUserResponseMessage); ok {
			bag := &entityv1.EntityBag{
				Users: []*entityv1.User{mapping.UserToProto(resp.User)},
			}

			if cities, err := h.srv.store.GetCitiesByOwner(ctx, claims.UserID); err == nil {
				for _, c := range cities {
					bag.Cities = append(bag.Cities, mapping.CityToProto(c))
					if buildings, err := h.srv.store.GetBuildingsByCity(ctx, c.CityID); err == nil {
						for _, b := range buildings {
							bag.Buildings = append(bag.Buildings, mapping.BuildingToProto(b))
						}
					}
				}
			}

			if err := out.Send(&servicev1.StreamStateResponse{Entities: bag}); err != nil {
				return err
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case update, ok := <-ch:
			if !ok {
				return nil
			}
			bag := &entityv1.EntityBag{}
			if update.User != nil {
				bag.Users = append(bag.Users, mapping.UserToProto(*update.User))
			}
			if update.City != nil {
				bag.Cities = append(bag.Cities, mapping.CityToProto(*update.City))
			}
			if update.Building != nil {
				bag.Buildings = append(bag.Buildings, mapping.BuildingToProto(*update.Building))
			}
			if err := out.Send(&servicev1.StreamStateResponse{Entities: bag}); err != nil {
				return err
			}
		}
	}
}
