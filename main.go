package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
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

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
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

func handlerAggregate(s *state, cmd command) error {
	_ = s
	if len(cmd.arguments) != 1 {
		fmt.Print("Wrong amount of arguments")
		os.Exit(1)
	}
	time_between_reqs := cmd.arguments[0]
	duration, err := time.ParseDuration(time_between_reqs)
	if err != nil {
		return err
	}
	fmt.Printf("Collecting feed every %s\n", time_between_reqs)
	ticker := time.NewTicker(duration)
	for ; ; <-ticker.C {
		scrapeFeed(s)
	}

}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.arguments) < 2 {
		os.Exit(1)
	}

	user_id := user.ID

	params := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.arguments[0],
		Url:       cmd.arguments[1],
		UserID:    user_id,
	}

	newFeed, err := s.db.CreateFeed(context.Background(), params)
	if err != nil {
		return err
	}
	fmt.Println(newFeed)
	params2 := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user_id,
		FeedID:    newFeed.ID,
	}
	rows, err := s.db.CreateFeedFollow(context.Background(), params2)
	if err != nil {
		return err
	}
	_ = rows
	return nil
}

func handlerListFeed(s *state, cmd command) error {
	if len(cmd.arguments) > 1 {
		os.Exit(1)
	}
	listFeedRow, err := s.db.ListFeed(context.Background())
	if err != nil {
		fmt.Println(err)
		return err
	}
	for _, feed := range listFeedRow {
		fmt.Printf("Name: %s, URL: %s, User: %s\n", feed.Name, feed.Url, feed.Name_2)
	}

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.arguments) > 2 {
		os.Exit(1)
	}
	feedId, err := s.db.GetFeed(context.Background(), cmd.arguments[0])
	if err != nil {
		fmt.Println("first err")
		return err
	}

	params := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feedId,
	}
	rows, err := s.db.CreateFeedFollow(context.Background(), params)
	if err != nil {
		fmt.Println("third err")
		return err
	}
	fmt.Println(rows)

	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	if len(cmd.arguments) > 1 {
		os.Exit(1)
	}
	following, err := s.db.GetFeedFollowsForUser(context.Background(), user.Name)
	if err != nil {
		return err
	}
	for _, follow := range following {
		fmt.Println(follow.Name_2)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.arguments) != 1 {
		os.Exit(1)
	}
	params := database.DeleteFeedFollowParams{
		ID:  user.ID,
		Url: cmd.arguments[0],
	}
	err := s.db.DeleteFeedFollow(context.Background(), params)
	if err != nil {
		return err
	}

	return nil
}

func handlerBrowse(s *state, cmd command) error {
	limit := int32(2)
	if len(cmd.arguments) > 0 {
		parsedLimit, err := strconv.Atoi(cmd.arguments[0])
		if err != nil {
			return err
		}
		limit = int32(parsedLimit)
	}

	params := database.GetPostsForUserParams{
		Name:  s.cfg.Username,
		Limit: limit,
	}
	posts, err := s.db.GetPostsForUser(context.Background(), params)
	if err != nil {
		return err
	}
	for _, post := range posts {
		fmt.Printf("ID: %v Created at: %v Updated at: %v Published at: %v\nTitle: %s Url: %s Description: %s\nFeed ID: %v\n",
			post.ID, post.CreatedAt, post.UpdatedAt, post.PublishedAt, post.Title, post.Url, post.Description.String, post.FeedID)
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

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return &RSSFeed{}, err
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return &RSSFeed{}, err
	}
	defer res.Body.Close()
	responseData, err := io.ReadAll(res.Body)
	if err != nil {
		return &RSSFeed{}, err
	}
	var response RSSFeed
	if err := xml.Unmarshal(responseData, &response); err != nil {
		return &RSSFeed{}, err
	}

	response.Channel.Title = html.UnescapeString(response.Channel.Title)
	response.Channel.Description = html.UnescapeString(response.Channel.Description)

	return &response, nil
}

func scrapeFeed(s *state) error {
	nextFeed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		fmt.Print("scrapeFeed: first err\n")
		return err
	}

	err = s.db.MarkFeedFetched(context.Background(), nextFeed.ID)
	if err != nil {
		fmt.Print("scrapeFeed: second err\n")
		return err
	}

	feedToFetch, err := fetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		fmt.Print("scrapeFeed: third err\n")
		return err
	}
	for _, item := range feedToFetch.Channel.Item {
		descriptionParam := sql.NullString{
			String: item.Description,
			Valid:  true,
		}
		timeParsed, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			fmt.Print("scrapeFeed: timeParsed err\n")
			return err
		}
		publishedAtParam := sql.NullTime{
			Time:  timeParsed,
			Valid: true,
		}
		params := database.CreatePostParams{
			ID:          uuid.New(),
			Title:       item.Title,
			Url:         item.Link,
			Description: descriptionParam,
			PublishedAt: publishedAtParam,
			FeedID:      nextFeed.ID,
		}
		newPost, err := s.db.CreatePost(context.Background(), params)
		if err != nil {
			fmt.Print("scrapeFeed: newPost err\n")
			fmt.Println(err)
			fmt.Printf("for feed %v\n", nextFeed.ID)
			return err
		}
		_ = newPost
		fmt.Print("post successfully added\n")
	}

	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.Username)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)
	}
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
	cmds.register("agg", handlerAggregate)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerListFeed)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.register("browse", handlerBrowse)
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
