package clicommands

import (
	"fmt"

	playlistutils "github.com/ItsOnlyGame/my-spotify-playlist-sorter-go/internal/utils/playlist-utils"
	urlutils "github.com/ItsOnlyGame/my-spotify-playlist-sorter-go/internal/utils/url-utils"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
	"github.com/zmb3/spotify/v2"
)

type DuplicateCommand struct {
	Sp *spotify.Client
}

func (cmd *DuplicateCommand) New() *cli.Command {
	return &cli.Command{
		Name:  "duplicate",
		Usage: "Duplicate a playlist",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "playlist",
				Usage:    "The playlist to duplicate",
				Aliases:  []string{"p"},
				Required: true,
			},
			&cli.StringFlag{
				Name:        "name",
				Usage:       "Name of the new playlist",
				Aliases:     []string{"n"},
				DefaultText: "Copy of <playlist name>",
				Required:    false,
			},
			&cli.BoolFlag{
				Name:        "public",
				Usage:       "Whether the new playlist should be public",
				DefaultText: "false",
				Required:    false,
			},
			&cli.BoolFlag{
				Name:        "collaborative",
				Usage:       "Whether the new playlist should be collaborative",
				DefaultText: "false",
				Required:    false,
			},
		},
		Action: cmd.Run,
	}
}

func (cmd *DuplicateCommand) Run(ctx *cli.Context) error {
	fmt.Printf("Duplicate command\n\n")

	playlistUrl := ctx.String("playlist")
	playlistId := spotify.ID(urlutils.ExtractSpotifyID(playlistUrl))

	// Get the playlist and its tracks to clone
	playlist, err := playlistutils.GetPlaylist(cmd.Sp, playlistId)

	if err != nil {
		return fmt.Errorf("failed to get playlist: %w", err)
	}

	tracks, err := playlistutils.GetPlaylistTracks(cmd.Sp, playlistId)

	if err != nil {
		return fmt.Errorf("failed to get playlist tracks: %w", err)
	}

	user, err := cmd.Sp.CurrentUser(ctx.Context)

	if err != nil {
		return fmt.Errorf("something went wront with setting up cloning operation: %w", err)
	}

	// Setting up variables to create the playlist
	newPlaylistName := ctx.String("name")
	if newPlaylistName == "" {
		newPlaylistName = "Copy of " + playlist.Name
	}

	description := "A copy of the playlist " + playlist.Name

	isPlaylistPublic := ctx.Bool("public")
	isPlaylistCollaborative := ctx.Bool("collaborative")

	// Create the new playlist
	clonedPlaylist, err := cmd.Sp.CreatePlaylistForUser(ctx.Context, user.ID, newPlaylistName, description, isPlaylistPublic, isPlaylistCollaborative)

	if err != nil {
		return fmt.Errorf("failed to create new playlist for user: %w", err)
	}

	bar := progressbar.Default(int64(len(tracks)), "Filtering out local tracks")
	filteredTracks := make([]*spotify.PlaylistItem, 0)
	for _, track := range tracks {
		bar.Add(1)
		if track.IsLocal {
			continue
		}
		filteredTracks = append(filteredTracks, track)
	}
	if len(tracks) != len(filteredTracks) {
		fmt.Printf("Filtered out %d local tracks \n", len(tracks)-len(filteredTracks))
	}

	// Add the tracks to the new playlist
	bar = progressbar.Default(int64(len(filteredTracks)), "Adding tracks to new playlist")
	for i := 0; i < len(filteredTracks); i += 100 {
		end := i + 100
		if end > len(filteredTracks) {
			end = len(filteredTracks)
		}

		trackIDs := make([]spotify.ID, len(filteredTracks[i:end]))
		for j, track := range filteredTracks[i:end] {
			if track.IsLocal {
				continue
			}

			trackIDs[j] = track.Track.Track.ID
		}

		_, err := cmd.Sp.AddTracksToPlaylist(ctx.Context, clonedPlaylist.ID, trackIDs...)
		bar.Add(len(trackIDs))
		if err != nil {
			return fmt.Errorf("failed to add tracks to new playlist: %w", err)
		}
	}

	fmt.Printf("Successfully cloned playlist: %s\n", newPlaylistName)

	return nil
}
