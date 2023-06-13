# execapi

execapi

## Start mongo server:

    docker run --rm --name mongo-main --network host -p 27017:27017 mongo

Test with mongo image:

    docker run -it --network host --rm mongo mongosh --eval 'show dbs;' mongodb://localhost:27017

## Start execapi:

    docker run -it --network host --rm -p 8080:8080 udhos/execapi:0.0.0

Test with execapi image:

    curl -d '{"cmd":["mongosh","--eval","show dbs","mongodb://localhost:27017"]}' localhost:8080/exec

Or loading command from file:

```
$ cat samples/sample1.yaml 
cmd:
  - mongosh
  - --eval
  - 'show dbs'
  - mongodb://localhost:27017

$ curl --data-binary @./samples/sample1.yaml localhost:8080/exec
```

## Docker images

https://hub.docker.com/r/udhos/execapi
