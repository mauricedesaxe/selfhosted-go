# Go on Rails

A simple framework (more template than framework) that helps you quickly develop
self-hosted web applications with Go.

## Things we like

- light client / hyper media focus (old school page reloads > HTMX > React)
- SQlite (portable, 0 latency db<->app, quite fast for self-hosted single-node web apps)
- Tailwind (easy to write, locality of behaviour)
- no NPM, no build JS, just simple JS scripts
- Templ templating language (multiple components per file, e2e type-safety)
- Docker (easy deploy on any server)
- [The Grug Brained Developer](https://grugbrain.dev/)

### Modular Design
We encourage organizing the application into distinct modules or domains. Domains should
represent business goals or sections of the site/application.

### Modified MVC Architecture
The framework adopts a modified version of the Model-View-Controller (MVC) architecture:

- **Controllers and Routes (`routes.go`)**: The core of every module. It holds the routes and their controllers / handlers.
It's supposed to export and `AddRoutes()` function to be used in `main.go`.
- **Models (`models.go`)**: When a module has many models / db tables / migrations, 
we separate them into a `models.go`. If a module has a simple db setup, we keep it in `routes.go`.
- **Views (`pages.templ` or `components.templ`)**: Views are managed through the Templ templating language. 
Wherever it makes sense we want to separate templ functions into pages, components and/or partials.

### Common utility features

- **Environment variables (`env.go`)**: We offer a global variable which can be accessed with `common.Env`. 
It uses struct tags to map environment variables and provide default values. This setup ensures that 
all necessary configurations are in place at runtime.
- **Mailer configuration (`mailer.go`)**: Offers an easy way to send emails. Stores the configuration
in SQlite instead of env variables. There are tradeoffs to this approach, but it suits self-hosted
applications well. For more info go to `mailer.go`.
- **Job Queue (`queue.go`)**: Helps schedule tasks to be processed async, such as sending emails. You're
supposed to create a new queue with its own workers and channel for each module where you need one. You can
then add jobs as you go. If a certain job name is defined as "lockable", then it can't be run concurrently.
This concurrency lock is useful in cases like: "I don't want to schedule a password reset email to the same user 3 times".
- **Components (`components.templ`)**: Base layouts, common pages, buttons, JS script invocation with built-in cache invalidation, 
HTMX (for ajax partials) and Quicklink (for prefetching) and other useful UI components to get you started.
- **Other utils (`utils.go`)**: Helps render templ templates, define caching rules, offers syntactic sugar like `TernaryIf()` or
`Jsonify()`, and other UI helpers.

There are other smaller utilities you may discover like the `Makefile` we wrote to help setup the project,
the `loaders.js` script to provide some interactivity cross-application when transitioning pages or 
Tailwind being pre-configured with the Tailwind CLI.

## How to setup for development

Make sure you have Go installed on your machine. We are using v1.21.3 right now.

Run the following commands:

```sh
# This command sets up a db folder, intalls templ, downloads tailwind CLI and inits tailwind.
# Use the version of tailwindcss CLI that you want, macos-arm64 is the default.
make setup ARCH="macos-arm64"

# Starts the development server (with file watcher) using air (https://github.com/cosmtrek/air).
# Air needs to be installed on your sistem to run this.
# The .air.toml config makes sure that templ and tailwind files are generated
# before it generates the binary and run it in dev environment.
make dev
```

That's it, you can start modifying code.

## How to deploy in production

Make sure you are on a machine that supports Docker and has the docker daemon runnning.

Use the standard docker compose commands, like:

```sh
docker compose up -d --build # build and start server
```