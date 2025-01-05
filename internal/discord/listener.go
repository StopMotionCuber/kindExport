package discord

import (
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"os/signal"
)

type Listener struct {
	session *discordgo.Session
}

func NewListener(token string) (Listener, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Printf("Error creating Discord session: %s", err.Error())
		return Listener{
			session: nil,
		}, err
	}
	// Init session
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err = session.Open()
	if err != nil {
		log.Printf("Error opening Discord session: %s", err.Error())
		return Listener{
			session: nil,
		}, err
	}
	return Listener{
		session: session,
	}, nil
}

func (l Listener) Listen() {
	err := initCommands(l.session)
	if err != nil {
		log.Printf("Error initializing commands: %s", err.Error())
		return
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Bot is running, use Ctrl+C to exit")
	<-stop

}
