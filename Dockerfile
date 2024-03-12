FROM golang:1.22

RUN apt-get update && apt-get install ffmpeg -y

WORKDIR modstreamapp

COPY [*.log] . .
RUN go mod tidy && go mod verify
RUN mkdir bin/ && go build -ldflags='-s -w' -o bin/modstream .

EXPOSE 8000/tcp
CMD ["bin/modstream"]