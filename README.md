# a9s RabbitMQ App

This is a sample app to check whether a RabbitMQ is working or not.
The application is a simple web server that is able to send messages to a
RabbitMQ queue and receive them. An internal map stores the received messages
and displays them on the main page. When you send a message it is possible you
have to reload the main page until the message was received from the queue
because of a small time lack caused by the network.

## Install, push and bind

Make sure you installed GO on your machine, [download this](https://golang.org/doc/install?download=go1.8.darwin-amd64.pkg) for mac.

Download the application
```
$ go get github.com/anynines/a9s-rabbitmq-app
$ cd $GOPATH/src/github.com/anynines/a9s-rabbitmq-app
```

Create a service on the [a9s PaaS](https://paas.anynines.com)
```
$ cf create-service a9s-rabbitmq36 rabbitmq-single-small myrabbitmq
```

Push the app
```
$ cf push --no-start
```

Bind the app
```
$ cf bind-service rabbitmq-app myrabbitmq
```

And start
```
$ cf start rabbitmq-app
```

At lsst check the created url...


## Local test

To start it locally you have to export the env variable VCAP_SERVICES
```
$ export VCAP_SERVICES='{
  "a9s-rabbitmq36": [
   {
    "credentials": {
     "host": "localhost",
     "password": "quest",
     "port": 5672,
     "uri": "amqp://guest:guest@localhost:5672",
     "username": "guest"
    }
   }
  ]
 }'
 ```

RabbitMQ Server with homebrew:
```shell
$ brew install rabbitmq
$ /usr/local/sbin/rabbitmq-server > rabbit.log &
```

Docker:
```shell
$ docker run -d -p 5672:5672 rabbitmq
```

Run the sample app
```
$ go build
$ ./a9s-rabbitmq-app
```

## Remark

To bind the app to other RabbitMQ services than `a9s-rabbitmq36`, have a look at the `VCAPServices` struct.
