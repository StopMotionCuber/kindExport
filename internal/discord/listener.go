package discord

import "github.com/bwmarrin/discordgo"

type Listener struct {
	session *discordgo.Session
}

func NewListener(token string) (Listener, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return Listener{}, err
	}
	return Listener{
		session: session,
	}, nil
}

func (l Listener) Listen() {

}
