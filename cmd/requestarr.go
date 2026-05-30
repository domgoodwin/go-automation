package cmd

import (
	"github.com/domgoodwin/go-automation/requestarr"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(requestarrCmd)

	requestarrCmd.Flags().String("sonarr-url", "", "Sonarr base URL (env: SONARR_URL)")
	requestarrCmd.Flags().String("sonarr-key", "", "Sonarr API key (env: SONARR_API_KEY)")
	requestarrCmd.Flags().String("radarr-url", "", "Radarr base URL (env: RADARR_URL)")
	requestarrCmd.Flags().String("radarr-key", "", "Radarr API key (env: RADARR_API_KEY)")
	requestarrCmd.Flags().String("addr", ":8080", "Listen address (env: REQUESTARR_ADDR)")
	requestarrCmd.Flags().String("tv-path", "", "Path to TV shows folder for on-disk check (env: TV_PATH)")
	requestarrCmd.Flags().String("movie-path", "", "Path to movies folder for on-disk check (env: MOVIE_PATH)")
	requestarrCmd.Flags().String("anime-path", "", "Path to anime folder (env: ANIME_PATH)")
	requestarrCmd.Flags().String("anime-quality-profile", "Any", "Sonarr quality profile name for anime (env: ANIME_QUALITY_PROFILE)")
	requestarrCmd.Flags().String("transmission-url", "", "Transmission base URL (env: TRANSMISSION_URL)")
	requestarrCmd.Flags().String("ll-url", "", "Lazy Librarian base URL (env: LL_URL)")
	requestarrCmd.Flags().String("books-path", "", "Path to books library root for on-disk lookup (env: BOOKS_PATH)")
	requestarrCmd.Flags().String("ll-key", "", "Lazy Librarian API key (env: LL_API_KEY)")
	requestarrCmd.Flags().String("kindle-store-path", "", "Directory for Kindle email and book request JSON files (env: KINDLE_STORE_PATH)")
	requestarrCmd.Flags().String("smtp-host", "", "SMTP host (env: SMTP_SERVER)")
	requestarrCmd.Flags().String("smtp-port", "587", "SMTP port (env: SMTP_PORT)")
	requestarrCmd.Flags().String("smtp-user", "", "SMTP username / from address (env: SMTP_USERNAME)")
	requestarrCmd.Flags().String("smtp-pass", "", "SMTP password / token (env: SMTP_TOKEN)")

	viper.BindPFlag("sonarr_url", requestarrCmd.Flags().Lookup("sonarr-url"))
	viper.BindPFlag("sonarr_api_key", requestarrCmd.Flags().Lookup("sonarr-key"))
	viper.BindPFlag("radarr_url", requestarrCmd.Flags().Lookup("radarr-url"))
	viper.BindPFlag("radarr_api_key", requestarrCmd.Flags().Lookup("radarr-key"))
	viper.BindPFlag("requestarr_addr", requestarrCmd.Flags().Lookup("addr"))
	viper.BindPFlag("tv_path", requestarrCmd.Flags().Lookup("tv-path"))
	viper.BindPFlag("movie_path", requestarrCmd.Flags().Lookup("movie-path"))
	viper.BindPFlag("anime_path", requestarrCmd.Flags().Lookup("anime-path"))
	viper.BindPFlag("anime_quality_profile", requestarrCmd.Flags().Lookup("anime-quality-profile"))
	viper.BindPFlag("transmission_url", requestarrCmd.Flags().Lookup("transmission-url"))
	viper.BindPFlag("ll_url", requestarrCmd.Flags().Lookup("ll-url"))
	viper.BindPFlag("books_path", requestarrCmd.Flags().Lookup("books-path"))
	viper.BindPFlag("ll_api_key", requestarrCmd.Flags().Lookup("ll-key"))
	viper.BindPFlag("kindle_store_path", requestarrCmd.Flags().Lookup("kindle-store-path"))
	viper.BindPFlag("smtp_host", requestarrCmd.Flags().Lookup("smtp-host"))
	viper.BindPFlag("smtp_port", requestarrCmd.Flags().Lookup("smtp-port"))
	viper.BindPFlag("smtp_user", requestarrCmd.Flags().Lookup("smtp-user"))
	viper.BindPFlag("smtp_pass", requestarrCmd.Flags().Lookup("smtp-pass"))

	viper.BindEnv("sonarr_url", "SONARR_URL")
	viper.BindEnv("sonarr_api_key", "SONARR_API_KEY")
	viper.BindEnv("radarr_url", "RADARR_URL")
	viper.BindEnv("radarr_api_key", "RADARR_API_KEY")
	viper.BindEnv("requestarr_addr", "REQUESTARR_ADDR")
	viper.BindEnv("tv_path", "TV_PATH")
	viper.BindEnv("movie_path", "MOVIE_PATH")
	viper.BindEnv("anime_path", "ANIME_PATH")
	viper.BindEnv("anime_quality_profile", "ANIME_QUALITY_PROFILE")
	viper.BindEnv("transmission_url", "TRANSMISSION_URL")
	viper.BindEnv("ll_url", "LL_URL")
	viper.BindEnv("books_path", "BOOKS_PATH")
	viper.BindEnv("ll_api_key", "LL_API_KEY")
	viper.BindEnv("kindle_store_path", "KINDLE_STORE_PATH")
	viper.BindEnv("smtp_host", "SMTP_SERVER")
	viper.BindEnv("smtp_port", "SMTP_PORT")
	viper.BindEnv("smtp_user", "SMTP_USERNAME")
	viper.BindEnv("smtp_pass", "SMTP_TOKEN")
}

var requestarrCmd = &cobra.Command{
	Use:   "requestarr",
	Short: "Web UI for requesting shows, movies, and books",
	Run: func(cmd *cobra.Command, args []string) {
		sonarrURL := viper.GetString("sonarr_url")
		sonarrKey := viper.GetString("sonarr_api_key")
		radarrURL := viper.GetString("radarr_url")
		radarrKey := viper.GetString("radarr_api_key")
		addr := viper.GetString("requestarr_addr")

		if sonarrURL == "" || sonarrKey == "" || radarrURL == "" || radarrKey == "" {
			log.Fatal("SONARR_URL, SONARR_API_KEY, RADARR_URL and RADARR_API_KEY are all required")
		}

		srv, err := requestarr.NewServer(requestarr.ServerConfig{
			SonarrURL:           sonarrURL,
			SonarrKey:           sonarrKey,
			RadarrURL:           radarrURL,
			RadarrKey:           radarrKey,
			TransmissionURL:     viper.GetString("transmission_url"),
			TVPath:              viper.GetString("tv_path"),
			MoviePath:           viper.GetString("movie_path"),
			AnimePath:           viper.GetString("anime_path"),
			AnimeQualityProfile: viper.GetString("anime_quality_profile"),
			LLURL:               viper.GetString("ll_url"),
			BooksPath:           viper.GetString("books_path"),
			LLKey:               viper.GetString("ll_api_key"),
			KindleStorePath:     viper.GetString("kindle_store_path"),
			SMTPHost:            viper.GetString("smtp_host"),
			SMTPPort:            viper.GetString("smtp_port"),
			SMTPUser:            viper.GetString("smtp_user"),
			SMTPPass:            viper.GetString("smtp_pass"),
		})
		if err != nil {
			log.Fatalf("failed to create server: %v", err)
		}
		if err := srv.Start(addr); err != nil {
			log.Fatalf("server error: %v", err)
		}
	},
}
