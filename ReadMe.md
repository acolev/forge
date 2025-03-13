# Forge CLI - Database Migrations Manager

## Overview

Forge CLI is a command-line tool designed to manage database migrations. It helps you apply and rollback migrations, ensuring your database schema is always up-to-date.

## Features

- **Apply Migrations**: Automatically apply new migrations to your database.
- **Rollback Migrations**: Rollback the last applied migration.
- **Environment Management**: Easily manage your environment variables with the `init` and `env` commands.
- **Plugin Support**: Load and register custom plugins to extend functionality.

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/acolev/forge.git
    cd forge
    ```

2. Install dependencies:
    ```sh
    go mod tidy
    ```

3. Build the project:
    ```sh
    go build -o forge cmd/main.go
    ```

## Usage

### Initialize Environment

Create or update the `.env` file with necessary environment variables:
```sh
forge init
```

Display Environment
Display the current environment variables:

```sh
forge env
```

Apply Migrations
Apply all pending migrations:

```sh
forge migrate
```

Rollback Last Migration
Rollback the last applied migration:

```sh
forge migrate:rollback
```

### Directory Structure
- **cmd/:** Contains the main entry point for the CLI.
- **pkg/migrations/:** Contains the migration logic.
- **pkg/plugins/:** Contains the plugin loading logic.

### Contributing
- Fork the repository.
- Create a new branch (`git checkout -b feature-branch`).
- Make your changes.
- Commit your changes (`git commit -am 'Add new feature'`).
- Push to the branch (`git push origin feature-branch`).
- Create a new Pull Request.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.