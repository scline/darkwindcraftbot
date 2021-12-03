package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// State - nested information
type State struct {
	ResourceAttributes struct {
		CurrentState string `json:"current_state"`
		IsSuspended  bool   `json:"is_suspended"`
		Resources    struct {
			MemoryBytes    int     `json:"memory_bytes"`
			CPU            float64 `json:"cpu_absolute"`
			DiskBytes      int     `json:"disk_bytes"`
			NetworkRXBytes int     `json:"network_rx_bytes"`
			NetworkTXBytes int     `json:"network_tx_bytes"`
		} `json:"resources"`
	} `json:"attributes"`
}

// Server - nested information
type Server struct {
	Attributes struct {
		UUID        string `json:"uuid"`
		Name        string `json:"name"`
		Node        string `json:"node"`
		Description string `json:"description"`
		Limits      struct {
			Memory int `json:"memory"`
			Swap   int `json:"swap"`
			Disk   int `json:"disk"`
			IO     int `json:"io"`
			CPU    int `json:"cpu"`
		} `json:"limits"`
		FeatureLimits struct {
			Databases   int `json:"databases"`
			Allocations int `json:"allocations"`
			Backups     int `json:"backups"`
		} `json:"feature_limits"`
		Relationships struct {
			Allocations struct {
				Data []struct {
					Attributes struct {
						Hostname string `json:"ip_alias"`
						Port     int    `json:"port"`
					} `json:"attributes"`
				} `json:"data"`
			} `json:"allocations"`
		} `json:"relationships"`
	} `json:"attributes"`
}

// Load environment variables from OS
func initEnvVars() map[string]string {

	// List of OS environment variables to load and store
	envs := []string{
		"API_URL",
		"API_KEY",
		"UUID_LIST",
		"DISCORD_TOKEN"}

	m := make(map[string]string)

	for _, env := range envs {
		m[env] = os.Getenv(env)
	}

	// Pass map of environment variables back to system
	return m
}

// HTTP GET function from admin API
func httpGET(url string, key string) []byte {

	// Create a Bearer string by appending string access token
	bearer := "Bearer " + key

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
	}

	// Set authorization headers
	req.Header.Add("Authorization", bearer)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	return body
}

// Load gameserver information
func getServerInfo(envs map[string]string, UUID string) (Server, State) {

	// Assign server information to struct from API
	serverJSON := httpGET(envs["API_URL"]+"/api/client/servers/"+UUID, envs["API_KEY"])

	var Server Server

	err := json.Unmarshal([]byte(serverJSON), &Server)
	if err != nil {
		log.Println(err)
	}

	// Assign server resource stats to struct from API
	stateJSON := httpGET(envs["API_URL"]+"/api/client/servers/"+UUID+"/resources", envs["API_KEY"])

	var State State

	err = json.Unmarshal([]byte(stateJSON), &State)
	if err != nil {
		log.Println(err)
	}

	return Server, State
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// TODO: get this out of the function or pass it directly somehow
	envs := initEnvVars()

	// Separate environment variable of gameserverID's to a list we can process
	UUIDs := strings.Split(envs["UUID_LIST"], ",")

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// If the message is "!status"
	if m.Content == "!status" {
		log.Println("!status - " + s.State.User.ID)
		for i := range UUIDs {
			Server, State := getServerInfo(envs, UUIDs[i])

			s.ChannelMessageSend(m.ChannelID, ">>> **"+Server.Attributes.Name+"** \n"+
				"_"+Server.Attributes.Description+"_\n"+
				"```\n"+
				"Server: "+Server.Attributes.Relationships.Allocations.Data[0].Attributes.Hostname+":"+strconv.Itoa(Server.Attributes.Relationships.Allocations.Data[0].Attributes.Port)+"\n"+
				"Status: "+State.ResourceAttributes.CurrentState+"\n"+
				"Resources:\n"+
				" * CPU:  "+strconv.FormatFloat(State.ResourceAttributes.Resources.CPU, 'f', 2, 64)+"%\n"+
				" * RAM:  "+strconv.Itoa(State.ResourceAttributes.Resources.MemoryBytes/1048576)+"MB/"+strconv.Itoa(Server.Attributes.Limits.Memory)+"MB\n"+
				" * DISK: "+strconv.Itoa(State.ResourceAttributes.Resources.DiskBytes/1048576)+"MB/"+strconv.Itoa(Server.Attributes.Limits.Disk)+"MB\n"+
				"```")
		}
	}

	// If the message is "!map"
	if m.Content == "!map" {
		log.Println("!map - " + s.State.User.ID)
		s.ChannelMessageSend(m.ChannelID, "Vanilla Online Map - https://map.legacy.darkwindcraft.com")
		s.ChannelMessageSend(m.ChannelID, "Legacy Online Map - https://map.vanilla.darkwindcraft.com")
	}

	// If the message is "!help"
	if m.Content == "!help" {
		log.Println("!help - " + s.State.User.ID)
		s.ChannelMessageSend(m.ChannelID, "_I'm a simple Discord bot that interacts with Darkwincraft Minecraft servers_\n"+
			"```==== Commands ===\n"+
			"!map    - Links to online Minecraft world maps\n"+
			"!status - Real time server status```")
	}
}

// Main program
func main() {
	log.Println("Starting up...")

	// Load environment variables
	envs := initEnvVars()

	discord, err := discordgo.New("Bot " + envs["DISCORD_TOKEN"])
	if err != nil {
		log.Println("error creating Discord session,", err)
		return
	}
	// Register the messageCreate func as a callback for MessageCreate events.
	discord.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	discord.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = discord.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	discord.Close()
}
