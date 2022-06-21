package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/maximtop/extdash/internal/chrome"
	"github.com/maximtop/extdash/internal/edge"
	"github.com/maximtop/extdash/internal/firefox"
	"github.com/urfave/cli/v2"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	chromeClient := chrome.Client{
		URL:          "https://accounts.google.com/o/oauth2/token",
		ClientID:     os.Getenv("CHROME_CLIENT_ID"),
		ClientSecret: os.Getenv("CHROME_CLIENT_SECRET"),
		RefreshToken: os.Getenv("CHROME_REFRESH_TOKEN"),
	}
	chromeStore, err := chrome.NewStore("https://www.googleapis.com")
	if err != nil {
		log.Fatal(err)
	}

	firefoxClient := firefox.NewClient(firefox.ClientConfig{
		ClientID:     os.Getenv("FIREFOX_CLIENT_ID"),
		ClientSecret: os.Getenv("FIREFOX_CLIENT_SECRET"),
	})
	firefoxStore, err := firefox.NewStore("https://addons.mozilla.org/")
	if err != nil {
		log.Fatal(err)
	}

	edgeClient, err := edge.NewClient(
		os.Getenv("EDGE_CLIENT_ID"),
		os.Getenv("EDGE_CLIENT_SECRET"),
		os.Getenv("EDGE_ACCESS_TOKEN_URL"),
	)
	if err != nil {
		log.Fatal(err)
	}

	edgeStore, err := edge.NewStore("https://api.addons.microsoftedge.microsoft.com")
	if err != nil {
		log.Fatal(err)
	}

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
			Usage: "returns extension info by id",
			Subcommands: []*cli.Command{
				{
					Name:  "firefox",
					Usage: "Firefox Store",
					Action: func(c *cli.Context) error {
						appID := c.String("app")

						status, err := firefoxStore.Status(firefoxClient, appID)
						if err != nil {
							return err
						}

						fmt.Println(status)

						return nil
					},
					Flags: []cli.Flag{appFlag},
				},
				{
					Name:  "chrome",
					Usage: "Chrome Store",
					Action: func(c *cli.Context) error {
						appID := c.String("app")

						status, err := chromeStore.Status(chromeClient, appID)
						if err != nil {
							return err
						}

						fmt.Println(status)

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
						filepath := c.String("file")

						result, err := chromeStore.Insert(chromeClient, filepath)
						if err != nil {
							return err
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
						filepath := c.String("file")
						sourcepath := c.String("source")

						err := firefoxStore.Insert(firefoxClient, filepath, sourcepath)
						if err != nil {
							return err
						}

						return nil
					},
				},
			},
		},
		{
			Name:  "update",
			Usage: "uploads new version of extension to the chrome web store",
			Subcommands: []*cli.Command{
				{
					Name:  "chrome",
					Usage: "updates version of extension in the chrome store",
					Flags: []cli.Flag{
						appFlag,
						fileFlag,
					},
					Action: func(c *cli.Context) error {
						filepath := c.String("file")
						appID := c.String("app")

						result, err := chromeStore.Update(chromeClient, appID, filepath)
						if err != nil {
							return err
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
						filepath := c.String("file")
						sourcepath := c.String("source")

						err := firefoxStore.Update(firefoxClient, filepath, sourcepath)
						if err != nil {
							return err
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
						filepath := c.String("file")
						appID := c.String("app")

						result, err := edgeStore.Update(edgeClient, appID, filepath, edge.UpdateOptions{})
						if err != nil {
							return err
						}

						fmt.Println(result)

						return nil
					},
				},
			},
		},
		{
			Name:  "publish",
			Usage: "publishes extension in the chrome web store",
			Subcommands: []*cli.Command{
				{
					Name:  "chrome",
					Usage: "publishes extension in the chrome store",
					Flags: []cli.Flag{
						appFlag,
					},
					Action: func(c *cli.Context) error {
						appID := c.String("app")

						result, err := chromeStore.Publish(chromeClient, appID)
						if err != nil {
							return err
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
						appID := c.String("app")

						result, err := edgeStore.Publish(edgeClient, appID)
						if err != nil {
							return err
						}

						fmt.Println(result)

						return nil
					},
				},
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
