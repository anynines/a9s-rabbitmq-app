package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/streadway/amqp"
)

type VCAP_Services struct {
	RabbitMQ []struct {
		Credentials struct {
			Host     string `json:"host"`
			Password string `json:"password"`
			Port     int    `json:"port"`
			URI      string `json:"uri"`
			Username string `json:"username"`
		} `json:"credentials"`
	} `json:"a9s-rabbitmq36"`
}

var rabbitMQUri string

type Message struct {
	Message    string
	ReceivedOn time.Time
}

var messageStore = make(map[string]Message)

var id int = 0

var templates map[string]*template.Template

func init() {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}
	templates["index"] = template.Must(template.ParseFiles("templates/index.html", "templates/base.html"))
	templates["new"] = template.Must(template.ParseFiles("templates/new.html", "templates/base.html"))

	var s = new(VCAP_Services)
	_ = json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &s)

	rabbitMQUri = (*s).RabbitMQ[0].Credentials.URI
}

func renderTemplate(w http.ResponseWriter, name string, template string, viewModel interface{}) {
	tmpl, _ := templates[name]
	err := tmpl.ExecuteTemplate(w, template, viewModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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

	r := mux.NewRouter().StrictSlash(false)
	fs := http.FileServer(http.Dir("public"))
	r.Handle("/public/", fs)
	r.HandleFunc("/", getMessages)
	r.HandleFunc("/messages/new", newMessage)
	r.HandleFunc("/messages/send", sendMessage)

	go startReceiver()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}
	log.Println("Listening...")
	server.ListenAndServe()
}
