package cmd

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/clbx/juicebot/util"

	// appsv1 "k8s.io/api/apps/v1"
	// corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Kubernetes client
var k8sClient *kubernetes.Clientset

// Check if guildID is in comma-separated list of guild IDs
func isGuildAuthorized(labelValue, guildID string) bool {
	guilds := strings.Split(labelValue, ",")
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
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Failed to connect to Kubernetes: %v", err),
				},
			})
			return
		}
	}

	var content string = "**Game Servers:**\n"
	guildID := i.GuildID

	// List all resources with the label, then filter by guild ID
	deployments, err := k8sClient.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "juicecloud.org/juicebot-game-server",
	})
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Failed to list deployments: %v", err),
			},
		})
		return
	}

	// List StatefulSets
	statefulSets, err := k8sClient.AppsV1().StatefulSets("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "juicecloud.org/juicebot-game-server",
	})
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Failed to list statefulsets: %v", err),
			},
		})
		return
	}

	// Filter and process Deployments by guild ID
	foundServers := false
	for _, deployment := range deployments.Items {
		// Check if this deployment belongs to the current guild
		labelValue, ok := deployment.Labels["juicecloud.org/juicebot-game-server"]
		if !ok || !isGuildAuthorized(labelValue, guildID) {
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
		labelValue, ok := statefulSet.Labels["juicecloud.org/juicebot-game-server"]
		if !ok || !isGuildAuthorized(labelValue, guildID) {
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
				Content: fmt.Sprintf("No game servers found for this guild. Make sure deployments/statefulsets have the label `juicecloud.org/juicebot-game-server` with this guild ID (%s) in the comma-separated list", guildID),
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
				Content: "Please specify a server ID to start (format: namespace/name)",
			},
		})
		return
	}

	serverID := options[0].StringValue()

	if k8sClient == nil {
		if err := initKubernetesClient(); err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Failed to connect to Kubernetes: %v", err),
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
				Content: "❌ Server ID must be in format: namespace/name",
			},
		})
		return
	}

	namespace, name := parts[0], parts[1]
	guildID := i.GuildID

	// Try to scale deployment first
	deployment, err := k8sClient.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		// Check if it has the required label and belongs to this guild
		labelValue, hasLabel := deployment.Labels["juicecloud.org/juicebot-game-server"]
		if !hasLabel || !isGuildAuthorized(labelValue, guildID) {
			if !hasLabel {
				log.Printf("User %s in guild %s attempted to access deployment %s/%s without juicebot label", i.Member.User.ID, guildID, namespace, name)
			} else {
				log.Printf("User %s in guild %s attempted to access resource %s/%s belonging to guilds %s", i.Member.User.ID, guildID, namespace, name, labelValue)
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
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Failed to start server: %v", err),
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
		labelValue, hasLabel := statefulSet.Labels["juicecloud.org/juicebot-game-server"]
		if !hasLabel || !isGuildAuthorized(labelValue, guildID) {
			if !hasLabel {
				log.Printf("User %s in guild %s attempted to access statefulset %s/%s without juicebot label", i.Member.User.ID, guildID, namespace, name)
			} else {
				log.Printf("User %s in guild %s attempted to access resource %s/%s belonging to guilds %s", i.Member.User.ID, guildID, namespace, name, labelValue)
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
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Failed to start server: %v", err),
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
				Content: "Please specify a server ID to stop (format: namespace/name)",
			},
		})
		return
	}

	serverID := options[0].StringValue()

	if k8sClient == nil {
		if err := initKubernetesClient(); err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Failed to connect to Kubernetes: %v", err),
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
				Content: "❌ Server ID must be in format: namespace/name",
			},
		})
		return
	}

	namespace, name := parts[0], parts[1]
	guildID := i.GuildID

	// Try to scale deployment first
	deployment, err := k8sClient.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		// Check if it has the required label and belongs to this guild
		labelValue, hasLabel := deployment.Labels["juicecloud.org/juicebot-game-server"]
		if !hasLabel || !isGuildAuthorized(labelValue, guildID) {
			if !hasLabel {
				log.Printf("User %s in guild %s attempted to access deployment %s/%s without juicebot label", i.Member.User.ID, guildID, namespace, name)
			} else {
				log.Printf("User %s in guild %s attempted to access resource %s/%s belonging to guilds %s", i.Member.User.ID, guildID, namespace, name, labelValue)
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
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Failed to stop server: %v", err),
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
		labelValue, hasLabel := statefulSet.Labels["juicecloud.org/juicebot-game-server"]
		if !hasLabel || !isGuildAuthorized(labelValue, guildID) {
			if !hasLabel {
				log.Printf("User %s in guild %s attempted to access statefulset %s/%s without juicebot label", i.Member.User.ID, guildID, namespace, name)
			} else {
				log.Printf("User %s in guild %s attempted to access resource %s/%s belonging to guilds %s", i.Member.User.ID, guildID, namespace, name, labelValue)
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
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Failed to stop server: %v", err),
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
