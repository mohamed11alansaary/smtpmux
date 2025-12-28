# SMTP Router Architecture

This document provides a detailed overview of the SMTP Router's architecture and implementation. The system is designed as a programmable SMTP proxy that accepts incoming emails, authenticates users, and dynamically routes emails to downstream SMTP servers based on user-defined logic.

## High-Level Overview

The SMTP Router acts as a middleware between an SMTP client (e.g., a mail sender) and multiple downstream SMTP providers (e.g., SendGrid, Mailgun, or internal relays). Its core value proposition is **dynamic routing**: instead of hardcoded failover logic, it uses an embedded Go interpreter (Yaegi) to execute routing algorithms defined in external files at runtime.

```mermaid
graph LR
    Client[SMTP Client] -->|SMTP :1025| Router[SMTP Router]
    Router -->|Load| Config[config.yaml]
    Router -->|Interpret| Algo[Routing Algo (e.g. round_robin.go)]
    Algo -->|Selects| DS1[Downstream 1]
    Algo -->|Selects| DS2[Downstream 2]
    Router -->|Forward| DS1
    Router -->|Forward| DS2
```

## Core Components

### 1. The Server (`main.go`)
The application entry point. It performs two main roles:
-   **Configuration Loading**: Reads `config.yaml` to load user credentials and downstream definitions.
-   **SMTP Listener**: Starts a TCP server on port 1025 using `github.com/emersion/go-smtp`.

### 2. Session Management
The `Session` struct is the heart of the request lifecycle. It implements the `smtp.Session` and `smtp.AuthSession` interfaces.

-   **State**: Holds the authenticated `UserConfig`, the `From` address, and a list of `To` recipients.
-   **Authentication**: Implements `AuthMechanisms` and `Auth` to support `PLAIN` authentication. It validates credentials against the loaded `config.yaml`.
-   **Envelope Handling**: The `Mail` and `Rcpt` methods capture the envelope sender and recipients, storing them in the session struct for later use.

### 3. Dynamic Routing Engine (Yaegi)
This is the most complex part of the system. When the email body is received (in the `Data` method), the router:
1.  Identifies the `selector_algo_path` for the authenticated user.
2.  Initializes a **Yaegi** interpreter (`github.com/traefik/yaegi`).
3.  Evaluates the external Go file (e.g., `round_robin.go`).
4.  Extracts the `SelectorAlgo` function from the interpreted code.
5.  Executes this function, passing in:
    -   The list of available `Downstreams`.
    -   A `sendFunc` closure that encapsulates the logic to actually send an email via a specific downstream.

### 4. Downstream Delivery
The `esendFunc` closure defined in `Data` is passed to the dynamic algorithm. When called by the algorithm, it:
-   Establishes a connection to the selected downstream.
-   Authenticates using the downstream's credentials.
-   Forwards the email using `net/smtp.SendMail`.

## Request Lifecycle

1.  **Connect**: Client connects to `:1025`.
2.  **EHLO**: Server advertises capabilities, including `AUTH PLAIN`.
3.  **AUTH**: Client sends credentials. `Session.Auth` validates them.
4.  **MAIL/RCPT**: Client sends sender and recipients. `Session` stores them.
5.  **DATA**: Client sends the email body.
    -   Server reads the body.
    -   Server loads the user's routing algorithm.
    -   Server executes the algorithm.
    -   Algorithm picks a downstream and calls `sendFunc`.
    -   `sendFunc` forwards the email.
6.  **Response**: Server returns `250 OK` to the client if forwarding succeeded.

## Configuration Structure

The `config.yaml` defines users and their associated downstreams.

```yaml
users:
  - email: "user@example.com"
    password: "password"
    selector_algo_path: "./round_robin.go" # Path to the dynamic algo
    downstreams:
      - addr: "smtp.provider1.com:587"
        user: "apikey"
        pass: "secret"
      - addr: "smtp.provider2.com:587"
        user: "user"
        pass: "pass"
```

## Extensibility

To add a new routing strategy (e.g., "Least Cost" or "Latency Based"), a user simply needs to:
1.  Write a new Go file with a `SelectorAlgo` function.
2.  Update their `selector_algo_path` in `config.yaml`.
3.  No recompilation of the main router is required.
