# dumbr

It is just as it states, a simple and dumb request/response handler capable of responding to incoming requests through response templates as configured up front. 

Typical usage is e.g. when you have the need to have a service for testing purposes always responding in the same way without having the backend code ready, call it a mockup or whatever. Or possibly you do not yet have a fully fledged kubernetes cluster capable of running hot deploys, hence you would like to have something dumb and simple to signal that you are now in maintenance mode.

## Building

Simply run go install ./...

## Configuring

### Configuring the response resources 

To which resources the server should be listening to can be configured through a json file as shown below or as found in the sample directory.



```json
{
    "requestConfig": [
        {
            "responseTemplateName": "helloWorld.template",
            "resource": "/resources/hello",
            "method": "GET"
        },
        {
            "responseTemplateName": "helloWorldPost.template",
            "resource": "/resources/hellopost",
            "method": "POST"
        }
    ]
}
```

* responseTemplateName : name of the golang text template to execute when hitting the configured resource
* resource : the resource url that should be valid for executing the template
* method : currently only POST or GET

The templating language is the standard as delivere together with Go. Templates will recieve the incoming *http.Request object meaning that data can be fetched directly from this in the template during execution.

E.g.

Printing all of the incoming http headers

```html
<table>
     
      {{range $key,$value := .Header}}
      <tr>
          <td>Header {{$key}}</td>
          <td>Value {{$value}}</td>
      </tr>
      {{end}}
</table>
```

### Configuring logging

dumbr uses the excellent uber zap go logging library. If a log configuration is specified as part of the input running parameters this will be used to configure the zap logging. If none is specified the default is stdout and info.

Example json log configuration found in the sample folder or as shown below.

```json
{
    "level": "debug",
    "encoding": "json",
    "outputPaths": [
        "stdout",
        "./log.txt"
    ],
    "errorOutputPaths": [
        "stderr"
    ],
    "encoderConfig": {
        "messageKey": "message",
        "levelKey": "level",
        "levelEncoder": "lowercase"
    }
}
```


## Running

Running is quite simple and dumb. Just use the example below in a terminal window where you have your compile dumbr binary.

Example: 

**./dumbr -templates=./templates -port="8080" -configuration=./sampleConfig.json -logconfig=./logConfig.json**


Supported input parameters

* templates -> path to the golang templates folder that contains all of your mapping response templates
* port -> yeah it is of course the port to listen to
* configuration -> path to the configuration file for templates and resources to map
* logconfig -> path to the uber/zap logging file
* serverCrt -> if you need to run TLS encryption you can put your certificate here
* serverKey -> and if you are running TLS encryption put your key for the certicate here


