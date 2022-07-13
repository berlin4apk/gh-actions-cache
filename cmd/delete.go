package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/actions/gh-actions-cache/internal"
	"github.com/actions/gh-actions-cache/service"
	"github.com/actions/gh-actions-cache/types"
	"github.com/spf13/cobra"
)

func NewCmdDelete() *cobra.Command {
	COMMAND = "delete"
	f := types.DeleteOptions{}

	var deleteCmd = &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete cache by key",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf(fmt.Sprintf("accepts 1 arg(s), received %d", len(args)))
			}

			f.Key = args[0]

			repo, err := internal.GetRepo(f.Repo)
			if err != nil {
				return err
			}

			artifactCache, err := service.NewArtifactCache(repo, COMMAND, VERSION)
			if err != nil {
				fmt.Printf("error connecting to %s\n", repo.Host())
				fmt.Println("check your internet connection or https://githubstatus.com")
				return nil
			}

			queryParams := url.Values{}
			f.GenerateBaseQueryParams(queryParams)

			if !f.Confirm {
				matchedCaches, err := getCacheListWithExactMatch(f, artifactCache)
				if err != nil {
					return err
				}
				matchedCachesLen := len(matchedCaches)
				if matchedCachesLen == 0 {
					return fmt.Errorf(fmt.Sprintf("Cache with input key '%s' does not exist\n", f.Key))
				}
				fmt.Printf("You're going to delete %s", internal.PrintSingularOrPlural(matchedCachesLen, "cache entry\n\n", "cache entries\n\n"))
				internal.PrettyPrintTrimmedCacheList(matchedCaches)
				choice := ""
				prompt := &survey.Select{
					Message: "Are you sure you want to delete the cache entries?",
					Options: []string{"Delete", "Cancel"},
				}
				err = survey.AskOne(prompt, &choice)
				if err != nil {
					return fmt.Errorf("Error occured while taking input from user while trying to delete cache")
				}
				f.Confirm = choice == "Delete"
				fmt.Println()
			}
			if f.Confirm {
				cachesDeleted, err := artifactCache.DeleteCaches(queryParams)
				if err != nil {
					return err
				}

				if cachesDeleted > 0 {
					fmt.Printf("%s Deleted %s with key '%s'\n", internal.RedTick(), internal.PrintSingularOrPlural(cachesDeleted, "cache entry", "cache entries"), f.Key)
				} else {
					fmt.Printf("Cache with input key '%s' does not exist\n", f.Key)
				}
			}
			return nil
		},
	}
	deleteCmd.Flags().StringVarP(&f.Repo, "repo", "R", "", "Select another repository for finding actions cache.")
	deleteCmd.Flags().StringVarP(&f.Branch, "branch", "B", "", "Filter by branch")
	deleteCmd.Flags().BoolVar(&f.Confirm, "confirm", false, "Delete the cache without asking user for confirmation.")
	deleteCmd.SetHelpTemplate(getDeleteHelp())

	return deleteCmd
}

func getDeleteHelp() string {
	return `
gh-actions-cache: Works with GitHub Actions Cache. 

USAGE:
	gh actions-cache delete <key> [flags]

ARGUMENTS:
	key		cache key which needs to be deleted
	
FLAGS:
	-R, --repo <[HOST/]owner/repo>		Select another repository using the [HOST/]OWNER/REPO format
	-B, --branch <string>			Filter by branch
	--confirm				Confirm deletion without prompting

INHERITED FLAGS
	--help		Show help for command

EXAMPLES:
	$ gh actions-cache delete Linux-node-f5dbf39c9d11eba80242ac13
`
}

func getCacheListWithExactMatch(f types.DeleteOptions, artifactCache service.ArtifactCacheService) ([]types.ActionsCache, error) {
	listOption := types.ListOptions{BaseOptions: types.BaseOptions{Repo: f.Repo, Branch: f.Branch, Key: f.Key}, Limit: 100, Order: "", Sort: ""}
	queryParams := url.Values{}

	listOption.GenerateBaseQueryParams(queryParams)
	caches, err := artifactCache.ListAllCaches(queryParams, f.Key)
	if err != nil {
		return nil, err
	}
	var exactMatchedKeys []types.ActionsCache
	for _, cache := range caches {
		if strings.EqualFold(f.Key, cache.Key) {
			exactMatchedKeys = append(exactMatchedKeys, cache)
		}
	}
	return exactMatchedKeys, nil
}