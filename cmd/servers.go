package cmd

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/clbx/juicebot/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var k8sClient *kubernetes.Clientset

// Check if guildID is in comma-separated list of guild IDs from annotations
func isGuildAuthorized(annotations map[string]string, guildID string) bool {
	annotationValue, exists := annotations["juicecloud.org/juicebot-guilds"]
	if !exists {
		return false
	}

	guilds := strings.Split(annotationValue, ",")
	for _, guild := range guilds {
		if strings.TrimSpace(guild) == guildID {
			return true
		}
	}
	return false
}

// Initialize Kubernetes client (kubeconfig first, then in-cluster)
func initKubernetesClient() error {
	var config *rest.Config
	var err error

	// Try kubeconfig first
	if home := homedir.HomeDir(); home != "" {
		kubeconfig := filepath.Join(home, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			// Fall back to in-cluster config
			config, err = rest.InClusterConfig()
			if err != nil {
				return fmt.Errorf("failed to create kubernetes config: %v", err)
			}
		}
	} else {
		// Try in-cluster config directly
		config, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("failed to create in-cluster kubernetes config: %v", err)
		}
	}

	k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	return nil
}

var ServersCommand = &discordgo.ApplicationCommand{
	Name:        "servers",
	Description: "Manage game servers",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "list",
			Description: "List all game servers",
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "start",
			Description: "Start a game server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "server",
					Description: "Server ID to start",
					Required:    true,
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "stop",
			Description: "Stop a game server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "server",
					Description: "Server ID to stop",
					Required:    true,
				},
			},
		},
	},
}

func ServersAction(s *discordgo.Session, i *discordgo.InteractionCreate, config *util.JuiceBotConfig) {
	options := i.ApplicationCommandData().Options

	if len(options) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Please specify a subcommand: list, start, or stop",
			},
		})
		return
	}

	subcommand := options[0]

	switch subcommand.Name {
	case "list":
		handleListServers(s, i)
	case "start":
		handleStartServer(s, i, subcommand.Options)
	case "stop":
		handleStopServer(s, i, subcommand.Options)
	}
}

func handleListServers(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if k8sClient == nil {
		if err := initKubernetesClient(); err != nil {
			log.Printf("Failed to initialize Kubernetes client for user %s in guild %s: %v", i.Member.User.ID, i.GuildID, err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Unable to connect to game servers",
				},
			})
			return
		}
	}

	var content string = "**Game Servers:**\n"
	guildID := i.GuildID

	// List all resources with the label in the games namespace, then filter by guild ID via annotations
	deployments, err := k8sClient.AppsV1().Deployments("games").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "juicecloud.org/juicebot-game-server=true",
	})
	if err != nil {
		log.Printf("Failed to list deployments for user %s in guild %s: %v", i.Member.User.ID, i.GuildID, err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Unable to retrieve game servers",
			},
		})
		return
	}

	// List StatefulSets
	statefulSets, err := k8sClient.AppsV1().StatefulSets("games").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "juicecloud.org/juicebot-game-server=true",
	})
	if err != nil {
		log.Printf("Failed to list statefulsets for user %s in guild %s: %v", i.Member.User.ID, i.GuildID, err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Unable to retrieve game servers",
			},
		})
		return
	}

	// Filter and process Deployments by guild ID
	foundServers := false
	for _, deployment := range deployments.Items {
		// Check if this deployment belongs to the current guild
		if !isGuildAuthorized(deployment.Annotations, guildID) {
			continue
		}
		foundServers = true

		statusEmoji := "🔴"
		status := "stopped"

		if deployment.Status.ReadyReplicas > 0 {
			statusEmoji = "🟢"
			status = "running"
		}

		serverName := deployment.Name
		if displayName, ok := deployment.Labels["app.kubernetes.io/name"]; ok {
			serverName = displayName
		}

		content += fmt.Sprintf("%s **%s** (%s/%s) - %s (%d/%d replicas)\n",
			statusEmoji, serverName, deployment.Namespace, deployment.Name, status,
			deployment.Status.ReadyReplicas, deployment.Status.Replicas)
	}

	// Filter and process StatefulSets by guild ID
	for _, statefulSet := range statefulSets.Items {
		// Check if this statefulSet belongs to the current guild
		if !isGuildAuthorized(statefulSet.Annotations, guildID) {
			continue
		}
		foundServers = true

		statusEmoji := "🔴"
		status := "stopped"

		if statefulSet.Status.ReadyReplicas > 0 {
			statusEmoji = "🟢"
			status = "running"
		}

		serverName := statefulSet.Name
		if displayName, ok := statefulSet.Labels["app.kubernetes.io/name"]; ok {
			serverName = displayName
		}

		content += fmt.Sprintf("%s **%s** (%s/%s) - %s (%d/%d replicas)\n",
			statusEmoji, serverName, statefulSet.Namespace, statefulSet.Name, status,
			statefulSet.Status.ReadyReplicas, statefulSet.Status.Replicas)
	}

	if !foundServers {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("No game servers found for this guild. Make sure deployments/statefulsets have the label `juicecloud.org/juicebot-game-server=true` and annotation `juicecloud.org/juicebot-guilds` containing this guild ID (%s)", guildID),
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func handleStartServer(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Please specify a server ID to start (format: games/name)",
			},
		})
		return
	}

	serverID := options[0].StringValue()

	if k8sClient == nil {
		if err := initKubernetesClient(); err != nil {
			log.Printf("Failed to initialize Kubernetes client for user %s in guild %s: %v", i.Member.User.ID, i.GuildID, err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Unable to connect to game servers",
				},
			})
			return
		}
	}

	// Parse namespace/name from serverID
	parts := strings.Split(serverID, "/")
	if len(parts) != 2 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Server ID must be in format: games/name",
			},
		})
		return
	}

	namespace, name := parts[0], parts[1]
	guildID := i.GuildID

	// Only allow operations in the games namespace
	if namespace != "games" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Server **%s** not found", serverID),
			},
		})
		return
	}

	// Try to scale deployment first
	deployment, err := k8sClient.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		// Check if it has the required label and belongs to this guild
		hasLabel := deployment.Labels["juicecloud.org/juicebot-game-server"] == "true"
		if !hasLabel || !isGuildAuthorized(deployment.Annotations, guildID) {
			if !hasLabel {
				log.Printf("User %s in guild %s attempted to access deployment %s/%s without juicebot label", i.Member.User.ID, guildID, namespace, name)
			} else {
				annotationValue := deployment.Annotations["juicecloud.org/juicebot-guilds"]
				log.Printf("User %s in guild %s attempted to access resource %s/%s belonging to guilds %s", i.Member.User.ID, guildID, namespace, name, annotationValue)
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Server **%s** not found", serverID),
				},
			})
			return
		}

		if *deployment.Spec.Replicas > 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Server **%s** is already running!", name),
				},
			})
			return
		}

		// Scale to 1 replica
		replicas := int32(1)
		deployment.Spec.Replicas = &replicas
		_, err = k8sClient.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("Failed to start deployment %s/%s for user %s in guild %s: %v", namespace, name, i.Member.User.ID, guildID, err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Unable to start server",
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("🟢 Starting server **%s** (%s)", name, serverID),
			},
		})
		return
	}

	// Try StatefulSet if deployment not found
	statefulSet, err := k8sClient.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		// Check if it has the required label and belongs to this guild
		hasLabel := statefulSet.Labels["juicecloud.org/juicebot-game-server"] == "true"
		if !hasLabel || !isGuildAuthorized(statefulSet.Annotations, guildID) {
			if !hasLabel {
				log.Printf("User %s in guild %s attempted to access statefulset %s/%s without juicebot label", i.Member.User.ID, guildID, namespace, name)
			} else {
				annotationValue := statefulSet.Annotations["juicecloud.org/juicebot-guilds"]
				log.Printf("User %s in guild %s attempted to access resource %s/%s belonging to guilds %s", i.Member.User.ID, guildID, namespace, name, annotationValue)
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Server **%s** not found", serverID),
				},
			})
			return
		}

		if *statefulSet.Spec.Replicas > 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Server **%s** is already running!", name),
				},
			})
			return
		}

		// Scale to 1 replica
		replicas := int32(1)
		statefulSet.Spec.Replicas = &replicas
		_, err = k8sClient.AppsV1().StatefulSets(namespace).Update(context.TODO(), statefulSet, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("Failed to start deployment %s/%s for user %s in guild %s: %v", namespace, name, i.Member.User.ID, guildID, err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Unable to start server",
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("🟢 Starting server **%s** (%s)", name, serverID),
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ Server **%s** not found", serverID),
		},
	})
}

func handleStopServer(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Please specify a server ID to stop (format: games/name)",
			},
		})
		return
	}

	serverID := options[0].StringValue()

	if k8sClient == nil {
		if err := initKubernetesClient(); err != nil {
			log.Printf("Failed to initialize Kubernetes client for user %s in guild %s: %v", i.Member.User.ID, i.GuildID, err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Unable to connect to game servers",
				},
			})
			return
		}
	}

	// Parse namespace/name from serverID
	parts := strings.Split(serverID, "/")
	if len(parts) != 2 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Server ID must be in format: games/name",
			},
		})
		return
	}

	namespace, name := parts[0], parts[1]
	guildID := i.GuildID

	// Only allow operations in the games namespace
	if namespace != "games" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Server **%s** not found", serverID),
			},
		})
		return
	}

	// Try to scale deployment first
	deployment, err := k8sClient.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		// Check if it has the required label and belongs to this guild
		hasLabel := deployment.Labels["juicecloud.org/juicebot-game-server"] == "true"
		if !hasLabel || !isGuildAuthorized(deployment.Annotations, guildID) {
			if !hasLabel {
				log.Printf("User %s in guild %s attempted to access deployment %s/%s without juicebot label", i.Member.User.ID, guildID, namespace, name)
			} else {
				annotationValue := deployment.Annotations["juicecloud.org/juicebot-guilds"]
				log.Printf("User %s in guild %s attempted to access resource %s/%s belonging to guilds %s", i.Member.User.ID, guildID, namespace, name, annotationValue)
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Server **%s** not found", serverID),
				},
			})
			return
		}

		if *deployment.Spec.Replicas == 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Server **%s** is already stopped!", name),
				},
			})
			return
		}

		// Scale to 0 replicas
		replicas := int32(0)
		deployment.Spec.Replicas = &replicas
		_, err = k8sClient.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("Failed to stop deployment %s/%s for user %s in guild %s: %v", namespace, name, i.Member.User.ID, guildID, err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Unable to stop server",
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("🔴 Stopping server **%s** (%s)", name, serverID),
			},
		})
		return
	}

	// Try StatefulSet if deployment not found
	statefulSet, err := k8sClient.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		// Check if it has the required label and belongs to this guild
		hasLabel := statefulSet.Labels["juicecloud.org/juicebot-game-server"] == "true"
		if !hasLabel || !isGuildAuthorized(statefulSet.Annotations, guildID) {
			if !hasLabel {
				log.Printf("User %s in guild %s attempted to access statefulset %s/%s without juicebot label", i.Member.User.ID, guildID, namespace, name)
			} else {
				annotationValue := statefulSet.Annotations["juicecloud.org/juicebot-guilds"]
				log.Printf("User %s in guild %s attempted to access resource %s/%s belonging to guilds %s", i.Member.User.ID, guildID, namespace, name, annotationValue)
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Server **%s** not found", serverID),
				},
			})
			return
		}

		if *statefulSet.Spec.Replicas == 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Server **%s** is already stopped!", name),
				},
			})
			return
		}

		// Scale to 0 replicas
		replicas := int32(0)
		statefulSet.Spec.Replicas = &replicas
		_, err = k8sClient.AppsV1().StatefulSets(namespace).Update(context.TODO(), statefulSet, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("Failed to stop deployment %s/%s for user %s in guild %s: %v", namespace, name, i.Member.User.ID, guildID, err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Unable to stop server",
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("🔴 Stopping server **%s** (%s)", name, serverID),
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ Server **%s** not found", serverID),
		},
	})
}
