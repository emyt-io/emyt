# Emty
Alternative Reverse Proxy for Python Django written in superb `go`

#### Why?
Nginx is too complicated to operate(for small deployments). I think of nginx as blackberry. It will be replaced one day IMO.

### Build

```
# If golang is missing then install go using the helper script.
$>sh setup_go.sh
```

```
Then build the Program
$>make default 

(or make osx, or make windows)

# All builds are located under `builds` folder
```

### Install (Only supported for Linux)

```
$>make install
```

Installation will create a file `/etc/emyt/app.env`.

Edit this file for the DOMAINS you wish to serve.

### Where are my logs?

Logs are located under `/var/log/emyt/emyt.log`

## Is it `Fast`?

I don't know, someone can run tests! But, its easier to configure than stupid nginx and gets the job done.
