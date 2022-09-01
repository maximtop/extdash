package main

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/joho/godotenv"
	"github.com/maximtop/extdash/internal/chrome"
	"github.com/maximtop/extdash/internal/edge"
	"github.com/maximtop/extdash/internal/firefox"
)

func getChromeStore() (*chrome.Store, error) {
	type config struct {
		ClientID     string `env:"CHROME_CLIENT_ID,notEmpty"`
		ClientSecret string `env:"CHROME_CLIENT_SECRET,notEmpty"`
		RefreshToken string `env:"CHROME_REFRESH_TOKEN,notEmpty"`
	}

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	client := chrome.Client{
		URL:          "https://accounts.google.com/o/oauth2/token",
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RefreshToken: cfg.RefreshToken,
	}

	store := chrome.Store{
		Client: &client,
		URL: &url.URL{
			Scheme: "https",
			Host:   "www.googleapis.com",
		},
	}

	return &store, nil
}

func getFirefoxStore() (*firefox.Store, error) {
	type config struct {
		ClientID     string `env:"FIREFOX_CLIENT_ID,notEmpty"`
		ClientSecret string `env:"FIREFOX_CLIENT_SECRET,notEmpty"`
	}

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	client := firefox.NewClient(firefox.ClientConfig{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
	})

	store := firefox.Store{
		Client: &client,
		URL: &url.URL{
			Scheme: "https",
			Host:   "addons.mozilla.org",
		},
	}

	return &store, nil
}

func getEdgeStore() (*edge.Store, error) {
	type config struct {
		ClientID     string `env:"EDGE_CLIENT_ID,notEmpty"`
		ClientSecret string `env:"EDGE_CLIENT_SECRET,notEmpty"`
		AccessToken  string `env:"EDGE_ACCESS_TOKEN,notEmpty"`
	}

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	client, err := edge.NewClient(
		cfg.ClientID,
		cfg.ClientSecret,
		cfg.AccessToken,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Edge Store Client: %w", err)
	}

	store := edge.Store{
		Client: &client,
		URL: &url.URL{
			Scheme: "https",
			Host:   "api.addons.microsoftedge.microsoft.com",
		},
	}

	return &store, nil
}

func main() {
	// we don't care if method fails on reading .env file, we will try to read config from environment
	// variables later
	_ = godotenv.Load()

	app := &cli.App{
		Name:  "extdash",
		Usage: "Cli application for managing extensions in the store",
	}

	appFlag := &cli.StringFlag{Name: "app", Aliases: []string{"a"}, Required: true}
	fileFlag := &cli.StringFlag{Name: "file", Aliases: []string{"f"}, Required: true}
	sourceFlag := &cli.StringFlag{Name: "source", Aliases: []string{"s"}, Required: true}

	app.Commands = []*cli.Command{
		{
			Name:  "status",
			Usage: "returns extension info",
			Subcommands: []*cli.Command{
				{
					Name:  "firefox",
					Usage: "Firefox Store",
					Action: func(c *cli.Context) error {
						store, err := getFirefoxStore()
						if err != nil {
							return fmt.Errorf("initializing firefox store: %w", err)
						}

						appID := c.String("app")

						status, err := store.Status(appID)
						if err != nil {
							return fmt.Errorf("getting status: %w", err)
						}

						fmt.Printf("%s\n", status)

						return nil
					},
					Flags: []cli.Flag{appFlag},
				},
				{
					Name:  "chrome",
					Usage: "Chrome Store",
					Action: func(c *cli.Context) error {
						store, err := getChromeStore()
						if err != nil {
							return fmt.Errorf("initializing chrome store: %w", err)
						}

						appID := c.String("app")
						status, err := store.Status(appID)
						if err != nil {
							return fmt.Errorf("getting status from chrome store: %w", err)
						}

						fmt.Printf("%s\n", status)

						return nil
					},
					Flags: []cli.Flag{appFlag},
				},
			},
		},
		{
			Name:  "insert",
			Usage: "uploads extension to the store",
			Subcommands: []*cli.Command{
				{
					Name:  "chrome",
					Usage: "inserts new extension to the chrome store",
					Flags: []cli.Flag{fileFlag},
					Action: func(c *cli.Context) error {
						store, err := getChromeStore()
						if err != nil {
							return fmt.Errorf("initializing chrome store: %w", err)
						}

						filepath := c.String("file")

						result, err := store.Insert(filepath)
						if err != nil {
							return fmt.Errorf("inserting extension: %w", err)
						}

						fmt.Println(result)

						return nil
					},
				},
				{
					Name:  "firefox",
					Usage: "inserts new extension to the firefox store",
					Flags: []cli.Flag{
						fileFlag,
						sourceFlag,
					},
					Action: func(c *cli.Context) error {
						store, err := getFirefoxStore()
						if err != nil {
							return fmt.Errorf("initializing firefox store: %w", err)
						}

						filepath := c.String("file")
						sourcepath := c.String("source")

						err = store.Insert(filepath, sourcepath)
						if err != nil {
							return fmt.Errorf("inserting extension: %w", err)
						}

						return nil
					},
				},
			},
		},
		{
			Name:  "update",
			Usage: "uploads new version of extension to the store",
			Subcommands: []*cli.Command{
				{
					Name:  "chrome",
					Usage: "updates version of extension in the chrome store",
					Flags: []cli.Flag{
						appFlag,
						fileFlag,
					},
					Action: func(c *cli.Context) error {
						store, err := getChromeStore()
						if err != nil {
							return fmt.Errorf("initializing chrome store: %w", err)
						}

						filepath := c.String("file")
						appID := c.String("app")

						result, err := store.Update(appID, filepath)
						if err != nil {
							return fmt.Errorf("updating extension: %w", err)
						}

						fmt.Println(result)

						return nil
					},
				},
				{
					Name:  "firefox",
					Usage: "updates version of extension in the firefox store",
					Flags: []cli.Flag{
						fileFlag,
						sourceFlag,
					},
					Action: func(c *cli.Context) error {
						store, err := getFirefoxStore()
						if err != nil {
							return fmt.Errorf("getting firefox store: %w", err)
						}

						filepath := c.String("file")
						sourcepath := c.String("source")

						err = store.Update(filepath, sourcepath)
						if err != nil {
							return fmt.Errorf("updating extension: %w", err)
						}

						return nil
					},
				},
				{
					Name:  "edge",
					Usage: "updates version of extension in the edge store",
					Flags: []cli.Flag{
						fileFlag,
						appFlag,
					},
					Action: func(c *cli.Context) error {
						store, err := getEdgeStore()
						if err != nil {
							return fmt.Errorf("getting edge store: %w", err)
						}

						filepath := c.String("file")
						appID := c.String("app")

						result, err := store.Update(appID, filepath, edge.UpdateOptions{})
						if err != nil {
							return fmt.Errorf("updating extension: %w", err)
						}

						fmt.Println(result)

						return nil
					},
				},
			},
		},
		{
			Name:  "publish",
			Usage: "publishes extension to the store",
			Subcommands: []*cli.Command{
				{
					Name:  "chrome",
					Usage: "publishes extension in the chrome store",
					Flags: []cli.Flag{
						appFlag,
					},
					Action: func(c *cli.Context) error {
						store, err := getChromeStore()
						if err != nil {
							return fmt.Errorf("initializing chrome store: %w", err)
						}

						appID := c.String("app")

						result, err := store.Publish(appID)
						if err != nil {
							return fmt.Errorf("publishing extension: %w", err)
						}

						fmt.Println(result)

						return nil
					},
				},
				{
					Name:  "edge",
					Usage: "publishes extension in the edge store",
					Flags: []cli.Flag{
						appFlag,
					},
					Action: func(c *cli.Context) error {
						store, err := getEdgeStore()
						if err != nil {
							return fmt.Errorf("getting edge store: %w", err)
						}

						appID := c.String("app")

						result, err := store.Publish(appID)
						if err != nil {
							return fmt.Errorf("publishing extension: %w", err)
						}

						fmt.Println(result)

						return nil
					},
				},
			},
		},
		{
			Name:  "sign",
			Usage: "signs extension in the store",
			Subcommands: []*cli.Command{
				{
					Name:  "firefox",
					Usage: "signs extension in the firefox store",
					Flags: []cli.Flag{
						fileFlag,
					},
					Action: func(c *cli.Context) error {
						store, err := getFirefoxStore()
						if err != nil {
							return fmt.Errorf("getting firefox store: %w", err)
						}

						filepath := c.String("file")

						err = store.Sign(filepath)
						if err != nil {
							return fmt.Errorf("signing extension: %w", err)
						}

						return nil
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("failed to run app: %s", err)
	}
}
