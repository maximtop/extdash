package main

import (
	"fmt"
	"log"
	"os"
)

import (
	"github.com/joho/godotenv"
	"github.com/maximtop/extdash/chrome"
	"github.com/maximtop/extdash/firefox"
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
	chromeStore := chrome.NewStore("https://www.googleapis.com")

	firefoxClient := firefox.Client{
		ClientID:     os.Getenv("FIREFOX_CLIENT_ID"),
		ClientSecret: os.Getenv("FIREFOX_CLIENT_SECRET"),
	}
	firefoxStore := firefox.NewStore("https://addons.mozilla.org/")

	app := &cli.App{
		Name:  "extdash",
		Usage: "Cli application for managing extensions in the store",
	}

	appFlag := &cli.StringFlag{Name: "app", Aliases: []string{"a"}, Required: true}
	fileFlag := &cli.StringFlag{Name: "file", Aliases: []string{"f"}, Required: true}

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
					Flags: []cli.Flag{fileFlag},
					Action: func(c *cli.Context) error {
						filepath := c.String("file")

						result, err := firefoxStore.Insert(firefoxClient, filepath)
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
					},
					Action: func(c *cli.Context) error {
						filepath := c.String("file")

						result, err := firefoxStore.Update(firefoxClient, filepath)
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
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "app", Aliases: []string{"a"}, Required: true},
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
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
