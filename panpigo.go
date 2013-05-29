package main

import (
	"bufio"
	"code.google.com/p/go.net/websocket"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const (
	defaultListenAddr = "localhost:4000" // default server address
)

var (
	prog          = "./pianobar/pianobar"
	pwd, _        = os.Getwd()
	RootTemp      = template.Must(template.ParseFiles(pwd + "/panpigo.html"))
	JSON          = websocket.JSON           // codec for JSON
	Message       = websocket.Message        // codec for string, []byte
	ActiveClients = make(map[ClientConn]int) // map containing clients
	listenAddr    = defaultListenAddr
	stationList   = map[string] string { "0" : "Waiting"}
	currentSong   = map[string] string {
		"artist" : "",
		"title" : "",
		"album" : "",
		"coverArt" : "",
		"stationName" : "",
	}
	piano_info    = make(chan string)
	piano_ctrl    = make(chan string)
)

// Initialize handlers and websocket handlers
func init() {
	if len(os.Args) > 1 {
		listenAddr = os.Args[1] //change default address and port using the command line
	}
	http.HandleFunc("/", RootHandler)
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("./js"))))
	http.Handle("/sock", websocket.Handler(SockServer))
	log.Println("Listening on", listenAddr)
}

// Client connection consists of the websocket and the client ip
type ClientConn struct {
	websocket *websocket.Conn
	clientIP  string
}

// WebSocket server to handle chat between clients
func SockServer(ws *websocket.Conn) {
	var err error
	var clientMessage string
	// use []byte if websocket binary type is blob or arraybuffer
	// var clientMessage []byte

	// cleanup on server side
	defer func() {
		if err = ws.Close(); err != nil {
			log.Println("Websocket could not be closed", err.Error())
		}
	}()

	client := ws.Request().RemoteAddr
	log.Println("Client connected:", client)
	sockCli := ClientConn{ws, client}
	ActiveClients[sockCli] = 0
	log.Println("Number of clients connected ...", len(ActiveClients))

	updateNewClient(sockCli)
	// for loop so the websocket stays open otherwise
	// it'll close after one Receive
	for {
		if err = Message.Receive(ws, &clientMessage); err != nil {
			// If we cannot Read then the connection is closed
			log.Println("Websocket Disconnected waiting", err.Error())
			// remove the ws client conn from our active clients
			delete(ActiveClients, sockCli)
			log.Println("Number of clients still connected ...", len(ActiveClients))
			return
		}
		piano_ctrl <- clientMessage
	}
}

func updateNewClient(cs ClientConn){
	for _, value := range currentSong {
		if err := Message.Send(cs.websocket, value); err != nil {
				// we could not send the message to a peer
				log.Println("Could not send message to ", cs.clientIP, err.Error())
		}
	}
	for _, value := range stationList {
		if err := Message.Send(cs.websocket, value); err != nil {
				// we could not send the message to a peer
				log.Println("Could not send message to ", cs.clientIP, err.Error())
		}
	}

}
// RootHandler renders the template for the root page
func RootHandler(w http.ResponseWriter, req *http.Request) {
	err := RootTemp.Execute(w, listenAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func processPianobarOutput(stdout io.Reader) {
	r := bufio.NewReader(stdout)
	for {
		chars, _, err := r.ReadLine()
		line := string(chars)
		if err == io.EOF {
			break
		}
		log.Print(line)
		myData := strings.Split(line, "\t")
		switch myData[0] {
			case "timerem":
				piano_info <- line
			case "current" :
				myCurrent := strings.Split(myData[1], "=")
				oldVal, ok := currentSong[ myCurrent[0] ]
				if ok && myCurrent[1] != oldVal {  //check for changes to current song
					currentSong[ myCurrent[0] ] = line
					piano_info <- line
				}
			case "station" :
				myStation := strings.Split(myData[1], "=")
				if myStation[0] != "stationCount" {
						station := strings.Replace(myStation[0], "station", "", -1)
						stationList[station] = line
				}
				//uncomment if you want to dynamically update clients when stationList changes
				//piano_info <- line  
		}
	}
}

func givePianobarInput(stdin io.Writer) {
	var input string
	for {
		input = <-piano_ctrl
		fmt.Println(input)
		myStrings := strings.Split(input, "\t")
		switch myStrings[0] {
			case "station" :
				fmt.Fprintf(stdin, "s%s\n", myStrings[1])
			case "command" :
				fmt.Fprintf(stdin, "%s", myStrings[1])
		}
	}
}

func notifyWebsocketClients() {
	for {
		message := <-piano_info
		for cs, _ := range ActiveClients {
			if err := Message.Send(cs.websocket, message); err != nil {
				// we could not send the message to a peer
				log.Println("Could not send message to ", cs.clientIP, err.Error())
			}
		}
	}
}

func main() {
	cmd := exec.Command(prog)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println(err)
	}
	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
	}
	go io.Copy(os.Stderr, stderr)
	go givePianobarInput(stdin)
	go processPianobarOutput(stdout)
	go notifyWebsocketClients()

	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
