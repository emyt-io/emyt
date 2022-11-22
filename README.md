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

## Security

As initial Security MVP we've implemented Basic HTTP authentication. To enable it per service you must include `use_auth: true` in the `app.yaml` file, and the traffic will be asked to to log in before reach the service. The session lasts aproximatelly 1 hour.

### Seeding users

To seed users you must provide a `users.json` file with the following structure:

```json
{
  "user_nodes": [
    {
      "username": "ernesto",
      "password": "s3cret"
    },
    {
      "username": "cj",
      "password": "SDTacos"
    },
    {
      "username": "nash",
      "password": "123Secret"
    }
  ]
}
```

Then run `$ go run cmd/main.go -seed`. Notice the `-seed` flag.

## Got Problems?

![Emyt Discord](https://discordapp.com/api/guilds/1042832870173052968/widget.png?style=banner3)

Or use link below

https://discord.gg/fB3FeFAvaU
