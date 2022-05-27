package main

import (
	"extdash/chrome"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	client := chrome.Client{
		URL:          "https://accounts.google.com/o/oauth2/token",
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		RefreshToken: os.Getenv("REFRESH_TOKEN"),
	}

	store := chrome.Store{URL: "https://www.googleapis.com"}

	app := &cli.App{
		Name:  "extdash",
		Usage: "Cli application for managing extensions in the store",
	}

	app.Commands = []*cli.Command{
		{
			Name:  "status",
			Usage: "returns extension info by id",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "app", Aliases: []string{"a"}, Required: true},
			},
			Action: func(c *cli.Context) error {
				appID := c.String("app")

				status, err := store.Status(client, appID)
				if err != nil {
					return err
				}

				fmt.Println(status)

				return nil
			},
		},
		{
			Name:  "insert",
			Usage: "uploads extension to the chrome web store",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Required: true},
			},
			Action: func(c *cli.Context) error {
				filepath := c.String("file")

				result, err := store.Insert(client, filepath)
				if err != nil {
					return err
				}

				fmt.Println(result)

				return nil
			},
		},
		{
			Name:  "update",
			Usage: "uploads new version of extension to the chrome web store",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Required: true},
				&cli.StringFlag{Name: "app", Aliases: []string{"a"}, Required: true},
			},
			Action: func(c *cli.Context) error {
				filepath := c.String("file")
				appID := c.String("app")

				result, err := store.Update(client, appID, filepath)
				if err != nil {
					return err
				}

				fmt.Println(result)

				return nil
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

				result, err := store.Publish(client, appID)
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
