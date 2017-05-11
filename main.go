package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/streadway/amqp"
)

// struct for reading env
type VCAPServices struct {
	RabbitMQ []struct {
		Credentials struct {
			Host     string `json:"host"`
			Password string `json:"password"`
			Port     int    `json:"port"`
			URI      string `json:"uri"`
			Username string `json:"username"`
		} `json:"credentials"`
	} `json:"a9s-rabbitmq"`
}

// store the URI to the rabbitmq
var rabbitMQUri string

// Message struct to store in the Map
type Message struct {
	Message    string
	ReceivedAt time.Time
}

// the map which stores the messages
var messageStore = make(map[string]Message)

// id counter
var id int = 0

// template store
var templates map[string]*template.Template

// fill template store and read env
func init() {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}
	templates["index"] = template.Must(template.ParseFiles("templates/index.html", "templates/base.html"))
	templates["new"] = template.Must(template.ParseFiles("templates/new.html", "templates/base.html"))

	// no new read of the env var, the reason is the receiver loop
	var s = new(VCAPServices)
	err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &s)
	if err != nil {
		log.Println(err)
		return
	}

	rabbitMQUri = (*s).RabbitMQ[0].Credentials.URI
}

func renderTemplate(w http.ResponseWriter, name string, template string, viewModel interface{}) {
	tmpl, _ := templates[name]
	err := tmpl.ExecuteTemplate(w, template, viewModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// the receiver for the RabbitMQ
func startReceiver() error {
	conn, err := amqp.Dial(rabbitMQUri)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("test-app", false, false, false, false, nil)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			message := Message{string(d.Body), time.Now()}

			id++
			k := strconv.Itoa(id)
			messageStore[k] = message
		}
	}()

	<-forever

	return nil
}

// send message to a RabbitMQ queue
func sendMessage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	conn, err := amqp.Dial(rabbitMQUri)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("test-app", false, false, false, false, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = ch.Publish("", q.Name, false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(r.PostFormValue("message")),
		})

	http.Redirect(w, r, "/", 302)
}

func newMessage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "new", "base", nil)
}

func getMessages(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index", "base", messageStore)
}

func main() {
	port := "9000"
	if port = os.Getenv("PORT"); len(port) == 0 {
		port = "9000"
	}

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir(path.Join(dir, "public")))
	http.Handle("/public/", http.StripPrefix("/public/", fs))
	http.HandleFunc("/", getMessages)
	http.HandleFunc("/messages/new", newMessage)
	http.HandleFunc("/messages/send", sendMessage)

	go startReceiver()

	log.Printf("Listening on port %v\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
