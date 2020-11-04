FROM golang:1.15

WORKDIR $GOPATH

ENV API_URL=https://something.com \
    UUID_LIST=abc-123,xyz-1452 \
    API_KEY=secret_key \
    DISCORD_TOKEN=secret_key

ADD src/ /src

RUN go get github.com/bwmarrin/discordgo

CMD [ "go", "run", "/src/main.go" ]
