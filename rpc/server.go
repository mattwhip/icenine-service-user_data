package rpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	dao "github.com/mattwhip/icenine-database/user_data"
	"github.com/mattwhip/icenine-service-user_data/config"
	"github.com/mattwhip/icenine-service-user_data/models"
	pb "github.com/mattwhip/icenine-services/generated/services/user_data"
	"github.com/gobuffalo/pop"
	"github.com/pkg/errors"
	glicko "github.com/zelenin/go-glicko2"
	"google.golang.org/grpc"
)

// Serve starts the GRPC server
func Serve() error {
	userDataServer := &Server{}
	grpcServer := grpc.NewServer()
	pb.RegisterUserDataServer(grpcServer, userDataServer)
	rpcListenPort := os.Getenv("RPC_LISTEN_PORT")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", rpcListenPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	return grpcServer.Serve(listener)
}

// Server implements protobuf generated UserDataServer interface.
type Server struct{}

// GetUser obtains data for a given user ID.
func (*Server) GetUser(ctx context.Context, req *pb.UserRequest) (*pb.UserResponse, error) {
	users := []dao.UdUser{}
	if err := models.DB.Where("u_id = ?", req.UID).All(&users); err != nil {
		return nil, errors.Wrapf(err, "failed to find user with userID %v", req.UID)
	}
	if len(users) == 0 {
		return &pb.UserResponse{
			Coins:  0,
			Exists: false,
			Rating: nil,
		}, nil
	}
	if len(users) > 1 {
		return nil, fmt.Errorf("found multiple users with UserID %v", req.UID)
	}
	return &pb.UserResponse{
		Coins:  users[0].Coins,
		Exists: true,
		Rating: &pb.Rating{
			Value:      users[0].Rating,
			Deviation:  users[0].RatingDeviation,
			Volatility: users[0].RatingVolatility,
		},
	}, nil
}

// InitNewUser initializes user data for a given user ID.
func (*Server) InitNewUser(ctx context.Context, req *pb.UserRequest) (*pb.UserResponse, error) {
	// Get config
	config, err := config.Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config")
	}
	// Create user data with initial config
	user := &dao.UdUser{
		UID:              req.UID,
		Coins:            config.InitialCoins,
		Rating:           config.InitialRating,
		RatingDeviation:  config.InitialRatingDeviation,
		RatingVolatility: config.InitialRatingVolatility,
	}
	if err := models.DB.Create(user); err != nil {
		return nil, errors.Wrapf(err, "failed to initialize new UserData user with UserID %v", req.UID)
	}
	return &pb.UserResponse{
		Coins:  user.Coins,
		Exists: true,
		Rating: &pb.Rating{
			Value:      user.Rating,
			Deviation:  user.RatingDeviation,
			Volatility: user.RatingVolatility,
		},
	}, nil
}

// DoCoinTransaction executes a coin transaction on one or more users
func (*Server) DoCoinTransaction(ctx context.Context, req *pb.CoinTransactionRequest) (*pb.CoinTransactionResponse, error) {
	// Get data for all users
	users := []dao.UdUser{}
	q := []string{}
	for uid := range req.Transactions {
		if len(q) > 0 {
			q = append(q, " or ")
		}
		q = append(q, fmt.Sprintf("u_id = '%s'", uid))
	}
	response := &pb.CoinTransactionResponse{
		Balances: make(map[string]int64),
	}
	if err := models.DB.Transaction(func(tx *pop.Connection) error {
		// Select and lock rows for users in request list
		rq := fmt.Sprintf("SELECT * FROM ud_users WHERE %s FOR UPDATE", strings.Join(q, ""))
		if err := tx.RawQuery(rq).All(&users); err != nil {
			return err
		}
		// Make sure data was found for every user ID sent with request
		if len(users) != len(req.Transactions) {
			return fmt.Errorf("Requested transactions for %d users but found user data for %d",
				len(req.Transactions), len(users))
		}
		// Update coin values for each user and build a response of final balances
		for i := range users {
			amount := req.Transactions[users[i].UID]
			if users[i].Coins+amount < 0 {
				return fmt.Errorf("Insufficient coins for user %s", users[i].UID)
			}
			users[i].Coins += amount
			response.Balances[users[i].UID] = users[i].Coins
		}
		// Save updates to database
		if err := tx.Update(users); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to execute coin transaction")
	}

	// Send response with updated balances
	return response, nil
}

// UpdateSkillLevels updates skill levels a group of players using a collection of match results
func (*Server) UpdateSkillLevels(ctx context.Context, req *pb.UpdateSkillRequest) (*pb.UpdateSkillResponse, error) {
	// Build list of all users involved in skill update request
	userIDMap := map[string]struct{}{}
	for _, matchResult := range req.MatchResults {
		userIDMap[matchResult.Player1] = struct{}{}
		userIDMap[matchResult.Player2] = struct{}{}
	}
	// Get data for all users
	users := []dao.UdUser{}
	q := []string{}
	for uid := range userIDMap {
		if len(q) > 0 {
			q = append(q, " or ")
		}
		q = append(q, fmt.Sprintf("u_id = '%s'", uid))
	}
	response := &pb.UpdateSkillResponse{
		Ratings: map[string]*pb.Rating{},
	}
	if err := models.DB.Transaction(func(tx *pop.Connection) error {
		// Select and lock rows for users in request list
		rq := fmt.Sprintf("SELECT * FROM ud_users WHERE %s FOR UPDATE", strings.Join(q, ""))
		if err := tx.RawQuery(rq).All(&users); err != nil {
			return err
		}

		// Build list of Glicko players using current rating data in DB
		players := map[string]*glicko.Player{}
		for _, user := range users {
			players[user.UID] = glicko.NewPlayer(glicko.NewRating(user.Rating, user.RatingDeviation, user.RatingVolatility))
		}

		// Calculate rating updates
		period := glicko.NewRatingPeriod()
		for _, matchResult := range req.MatchResults {
			p1 := players[matchResult.Player1]
			p2 := players[matchResult.Player2]
			period.AddMatch(p1, p2, glicko.MatchResult(matchResult.Score))
		}
		period.Calculate()

		// Apply rating updates to user DAOs
		for _, user := range users {
			player := players[user.UID]
			user.Rating = player.Rating().R()
			user.RatingDeviation = player.Rating().Rd()
			user.RatingVolatility = player.Rating().Sigma()
		}

		// Add updated rating values to response object
		for _, user := range users {
			player := players[user.UID]
			response.Ratings[user.UID] = &pb.Rating{
				Value:      player.Rating().R(),
				Deviation:  player.Rating().Rd(),
				Volatility: player.Rating().Sigma(),
			}
		}

		// Save updates to database
		if err := tx.Update(users); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to execute coin transaction")
	}

	// Send response with updated skill ratings
	return response, nil
}

// GetBalances retrieves balances for one or more users
func (s *Server) GetBalances(ctx context.Context, req *pb.BalancesRequest) (*pb.BalancesResponse, error) {
	// Get data for all users
	users := []dao.UdUser{}
	q := []string{}
	for _, uid := range req.UserIDs {
		if len(q) > 0 {
			q = append(q, " or ")
		}
		q = append(q, fmt.Sprintf("u_id = '%s'", uid))
	}
	response := &pb.BalancesResponse{
		Balances: make(map[string]int64),
	}
	if err := models.DB.Transaction(func(tx *pop.Connection) error {
		// Select rows for users in request list
		rq := fmt.Sprintf("SELECT * FROM ud_users WHERE %s", strings.Join(q, ""))
		if err := tx.RawQuery(rq).All(&users); err != nil {
			return err
		}
		// Make sure data was found for every user ID sent with request
		if len(users) != len(req.UserIDs) {
			return fmt.Errorf("Requested balances for %d users but found user data for %d",
				len(req.UserIDs), len(users))
		}
		// Build a response of final balances
		for _, user := range users {
			response.Balances[user.UID] = user.Coins
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to execute get balances transaction")
	}

	// Send response with balances
	return response, nil
}
