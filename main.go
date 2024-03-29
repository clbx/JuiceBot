package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"

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

	flag.Parse()
}

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	config = *util.NewJuiceBotConfig(*ConfigPath)

	if config.Debug {
		fmt.Printf("%+v\n", config)
	}
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
	}

	// Add commands here.
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ping(s, i) // Assuming 'ping' is adjusted to accept JuiceBotConfig
		},
		"dog": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			cmd.DogAction(s, i, &config) // Adjust 'DogAction' to accept JuiceBotConfig
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
}

func main() {

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	if *RemoveCommands {
		log.Println("Removing commands...")
		// // We need to fetch the commands, since deleting requires the command ID.
		// // We are doing this from the returned commands on line 375, because using
		// // this will delete all the commands, which might not be desirable, so we
		// // are deleting only the commands that we added.
		// registeredCommands, err := s.ApplicationCommands(s.State.User.ID, *GuildID)
		// if err != nil {
		// 	log.Fatalf("Could not fetch registered commands: %v", err)
		// }

		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, *GuildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	log.Println("Gracefully shutting down.")
}
