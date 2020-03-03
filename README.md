# a9s RabbitMQ App

This is a sample application to check whether a RabbitMQ server is working or not.
The application is a simple web server that is able to send messages to a
RabbitMQ queue and receive them. An internal map stores the received messages
and displays them on the main page. When you send a message you might have
to reload the main page until the message was received from the queue
because of a small time lack caused by the network.

## Install, push and bind

Make sure you installed Go on your machine, [download this](https://golang.org/doc/install?download=go1.8.darwin-amd64.pkg) for mac.

Download the application
```
$ go get github.com/anynines/a9s-rabbitmq-app
$ cd $GOPATH/src/github.com/anynines/a9s-rabbitmq-app
```

Create a service on the [a9s PaaS](https://paas.anynines.com)
```
$ cf create-service a9s-rabbitmq36 rabbitmq-single-small myrabbitmq
```

Build the app locally
```
$  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 make
```

Push the app
```
$ cf push --no-start
```

Bind the app
```
$ cf bind-service rabbitmq-app myrabbitmq
```

**Known Issues**
It is possible the application is not starting and you find the following in your app logs.
 &nbsp;
```bash
$cf logs rabbitmq-app --recent
Retrieving logs for app rabbitmq-app in org training / space phartz as phartz@anynines.com...

   2019-02-19T09:04:23.02+0100 [API/1] OUT Created app with guid 09c5d595-d5f3-41b5-9175-1e2d7fc114b0
   ...
   2019-02-19T09:15:43.48+0100 [APP/PROC/WEB/0] ERR 2019/02/19 08:15:43 no valid service instance was found; specify SERVICE_INSTANCE_NAME or ensure "rabbitmq" tag is present
   2019-02-19T09:15:43.48+0100 [APP/PROC/WEB/0] OUT Exit status 1
```
See the [configuration section](#configuration) of this document.
&nbsp;

And start
```
$ cf start rabbitmq-app
```

At last check the created url...


## Local test

To start it locally you have to export the env variable VCAP_SERVICES
```
$ export VCAP_SERVICES='{
  "a9s-rabbitmq36": [
   {
    "tags": [ "rabbitmq" ],
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

## Configuration

The app will bind to the first service instance that contains the `rabbitmq`
tag, or you can set the `SERVICE_INSTANCE_NAME` variable to specify the service
instance you would like to use.
See the example below how to do that in the `manifest.yml`.

```yaml
applications:
- name: rabbitmq-app
  memory: 128M
  random-route: true
  instances: 1
  path: .
  buildpack: binary_buildpack
  command: ./rabbitmq-app
  env:
    GOPACKAGENAME: rabbitmq-app
    SERVICE_INSTANCE_NAME: myrabbitmq
```
