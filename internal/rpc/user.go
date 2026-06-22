package rpc

import (
	"context"
	"errors"
	"strings"

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

const minPasswordLength = 8

type userHandler struct {
	srv *Server
}

func (h *userHandler) Register(ctx context.Context, req *connect.Request[servicev1.RegisterRequest]) (*connect.Response[servicev1.RegisterResponse], error) {
	email := strings.TrimSpace(req.Msg.GetEmail())
	username := strings.TrimSpace(req.Msg.GetUsername())
	password := req.Msg.GetPassword()

	if email == "" || username == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email and username are required"))
	}
	if len(password) < minPasswordLength {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("password must be at least 8 characters"))
	}

	// Surface duplicates with AlreadyExists before incurring the actor spawn +
	// bcrypt cost. The DB still has the UNIQUE constraint as a backstop.
	if existing, err := h.srv.store.GetUserByIdentifier(ctx, email); err == nil && existing != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("email already registered"))
	} else if err != nil && !errors.Is(err, persistence.ErrNotFound) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if existing, err := h.srv.store.GetUserByIdentifier(ctx, username); err == nil && existing != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("username already taken"))
	} else if err != nil && !errors.Is(err, persistence.ErrNotFound) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	userID, err := services.CreateUser(ctx, h.srv.cluster, &services.CreateUserRequest{
		Email:    email,
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	token, err := auth.Issue(h.srv.jwtSecret, auth.Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
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

			if dbCities, err := h.srv.store.GetCitiesByOwner(ctx, claims.UserID); err == nil {
				for _, dc := range dbCities {
					if res, err := h.srv.cluster.Request("city", dc.CityID, messages.GetCityMessage{}); err == nil {
						if cr, ok := res.(*messages.GetCityResponseMessage); ok {
							bag.Cities = append(bag.Cities, mapping.CityToProto(cr.City))
						}
					}
					if dbBuildings, err := h.srv.store.GetBuildingsByCity(ctx, dc.CityID); err == nil {
						for _, db := range dbBuildings {
							if res, err := h.srv.cluster.Request("building", db.BuildingID, messages.GetBuildingMessage{}); err == nil {
								if br, ok := res.(*messages.GetBuildingResponseMessage); ok {
									bag.Buildings = append(bag.Buildings, mapping.BuildingToProto(br.Building))
								}
							}
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
		case <-h.srv.shutdownCtx.Done():
			// Server is shutting down. Return Unauthenticated so the client's
			// auth-error path runs (clears JWT, redirects to /login) — same
			// shape it would see if the JWT had expired mid-session.
			return connect.NewError(connect.CodeUnauthenticated, errors.New("server shutting down"))
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
			if update.DeletedBuildingID != nil {
				bag.DeletedBuildingIds = append(bag.DeletedBuildingIds, mapping.ToBuildingId(*update.DeletedBuildingID))
			}
			if err := out.Send(&servicev1.StreamStateResponse{Entities: bag}); err != nil {
				return err
			}
		}
	}
}
