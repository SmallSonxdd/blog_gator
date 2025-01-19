package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/lib/pq"

	config "github.com/SmallSonxdd/blog_gator/internal/config"
	"github.com/SmallSonxdd/blog_gator/internal/database"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name      string
	arguments []string
}

type commands struct {
	command map[string]func(*state, command) error
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.arguments) == 0 {
		return fmt.Errorf("not enough arguments")
	}
	username := cmd.arguments[0]

	user, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {

			fmt.Println("Error: User does not exist")
			os.Exit(1)
		}

		return fmt.Errorf("failed to check user existence: %w", err)
	}

	err = s.cfg.SetUser(cmd.arguments[0])
	if err != nil {
		return err
	}

	fmt.Printf("User %v has been set\n", user.Name)
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.arguments) == 0 {
		return fmt.Errorf("not enough arguments")
	}
	params := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.arguments[0],
	}

	_, err := s.db.CreateUser(context.Background(), params)
	if err != nil {
		log.Printf("Error occurred during CreateUser: %v", err)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			os.Exit(1)
		}
		return fmt.Errorf("database error: %w", err)
	}
	log.Println("User creation was called successfully!")

	s.cfg.SetUser(cmd.arguments[0])
	fmt.Println("User has been created:")
	fmt.Println(params)

	return nil
}

func handlerDeleteUsers(s *state, cmd command) error {
	if err := s.db.DeleteAllUsers(context.Background()); err != nil {
		fmt.Println("Truncation unsuccessful. Better luck next time!")
		return err
	}
	fmt.Println("Successfully truncated!")
	return nil
}

func handlerGetUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Println(err)
		return err
	}
	if len(users) == 0 {
		fmt.Println("there are no users in the database")
		os.Exit(0)
	}
	currentUser := s.cfg.Username
	for _, user := range users {
		lineMessage := fmt.Sprintf("* %s", user)
		if user == currentUser {
			lineMessage += " (current)"
		}
		fmt.Println(lineMessage)
	}

	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.command[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	comName, ok := c.command[cmd.name]
	if !ok {
		return fmt.Errorf("this command does not exist")
	}
	if err := comName(s, cmd); err != nil {
		return err
	}
	return nil
}

func main() {

	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/gator")
	if err != nil {
		fmt.Println(err)
	}
	dbQueries := database.New(db)

	st := state{}
	cfg, err := config.Read()
	if err != nil {
		fmt.Println(err)
	}
	st.cfg = &cfg
	st.db = dbQueries

	cmds := commands{command: make(map[string]func(*state, command) error)}
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerDeleteUsers)
	cmds.register("users", handlerGetUsers)
	params := os.Args
	if _, ok := cmds.command[params[1]]; !ok {
		fmt.Printf("What's up %s\nThere is no such command\n", params[1])
		os.Exit(0)
	}
	if len(params) > 2 {
		cmd := command{name: params[1],
			arguments: params[2:],
		}
		cmds.run(&st, cmd)
	} else {
		cmd := command{name: params[1],
			arguments: []string{},
		}
		cmds.run(&st, cmd)
	}

}
