package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/clbx/juicebot/cmd"
	"github.com/clbx/juicebot/util"
)

var config util.JuiceBotConfig

// Bot parameters
var (
	GuildID        = flag.String("guild", "", "Test guild ID. If not passed - bot registers commands globally")
	BotToken       = flag.String("token", "", "Bot access token")
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
	ConfigPath     = flag.String("config", "./config.yaml", "Path to the config file")
)

var s *discordgo.Session
var db *sql.DB

func init() {
	tokenEnv := os.Getenv("TOKEN")
	if tokenEnv != "" {
		log.Printf("Token loaded from Environment Variable")
		BotToken = &tokenEnv
	}

	configEnv := os.Getenv("CONFIG")
	if configEnv != "" {
		log.Printf("Config path loaded from Environment Variable")
		ConfigPath = &configEnv
	}

	postgresConfigEnv := os.Getenv("POSTGRES_URI")
	if postgresConfigEnv != "" {
		log.Printf("Postgres URI Loaded")
	}

	//init db
	var err error
	db, err = sql.Open("pgx", postgresConfigEnv)
	if err != nil {
		log.Fatalf("Failed to connect to database, %W", err)
	}
	util.InitDB(db)

	flag.Parse()
}

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	s.Identify.Intents = discordgo.IntentsGuildMembers | discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent
	s.State.TrackMembers = true

	config = *util.NewJuiceBotConfig(*ConfigPath)

	if config.Debug {
		fmt.Printf("%+v\n", config)
	}

	log.Printf("Bot intents set to: %d", s.Identify.Intents)
}

var (
	// integerOptionMinValue          = 1.0
	// dmPermission                   = false
	// defaultMemberPermissions int64 = discordgo.PermissionManageServer

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "pong",
		},
		cmd.DogCommand,
		cmd.ServersCommand,
		cmd.NameHistoryCommand,
	}

	// Add commands here.
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ping(s, i)
		},
		"dog": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			cmd.DogAction(s, i, &config)
		},
		"servers": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			cmd.ServersAction(s, i, &config)
		},
		"namehistory": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			cmd.NameHistoryAction(s, i, &config, db)
		},
	}
)

func ping(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "pong!",
		},
	})
}

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	// Add Handlers here.
	// s.AddHandler(cmd.DogHandler)
	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		cmd.DogHandler(s, m, &config)
	})

	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		cmd.CalloutHandler(s, m, &config)
	})

	s.AddHandler(func(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
		log.Printf("GuildMemberUpdate event received - User: %s, Guild: %s, Nick: %s", m.User.ID, m.GuildID, m.Nick)
		if m.BeforeUpdate != nil {
			log.Printf("  Before: Nick=%s", m.BeforeUpdate.Nick)
		}
		cmd.NameHistoryHandler(s, m, &config, db)
	})

}

func main() {

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		log.Printf("Ready event - Guilds: %d", len(r.Guilds))
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	log.Println("Adding commands...")
	log.Printf("%d Commands found\n", len(commands))
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		log.Printf("Registered %v\n", v.Name)
		registeredCommands[i] = cmd
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	if *RemoveCommands {
		log.Println("Removing commands...")
		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, *GuildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
			log.Printf("Removed %v\n", v.Name)
		}

	}

	log.Println("Gracefully shutting down.")
}
