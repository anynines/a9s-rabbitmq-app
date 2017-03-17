# A9s-RabbitMQ-App

This is a sample app to check whether a RabbitMQ is working or not.
The application is a simple WebServer which is able to send a message to a RabbitMQ queue and receives them.
An internal map stores the received messages and diyplays them on the main page. When you send a message it is possible you have to reload the main page until the message was received from the queue because of a small time lack caused by the network.

## Install, push and bind

Download the application
```
go get github.com/phartz/a9s-rabbitmq-app
cd $GOPATH/src/github.com/phartz/a9s-rabbitmq-app
```

Create a service on the [a9s PAAS](https://paas.anynines.com)
```
cf create-service a9s-rabbitmq36 rabbitmq-single-small myrabbitmq
```

Push the app
```
cf push --no-start
```

Bind the app
```
cf bind-service rabbitmq-app myrabbitmq
```

And restage
```
cf restage rabbitmq-app
```

At least check the created url...


## Local test

To start it locally you have to export the env variable VCAP_SERVICES
```
export VCAP_SERVICES='{
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

You can install a RabbitMQ Server with homebrew:
```
brew install rabbitmq
```

Now start a local RabbitMQ in Background
```
/usr/local/sbin/rabbitmq-server > rabbit.log &
```

At least run the sample app
```
go run main.go
```
