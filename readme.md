# DataTracker: Social Media Analytics Backend

A robust, scalable, and efficient backend for analyzing social media data, built with Go and Firebase Firestore. This platform is designed to handle high write loads while ensuring data consistency and reliability across a distributed architecture.

## Table of Contents

- [Project Overview](#project-overview)
- [Core Architecture & Technologies](#core-architecture--technologies)
- [Key Design Concepts](#key-design-concepts)
  - [Replication Strategies](#replication-strategies)
  - [Consistency Models](#consistency-models)
  - [Advanced Data Structures](#advanced-data-structures)
- [Installation and Setup](#installation-and-setup)
- [Running the Application](#running-the-application)
- [Monitoring & Operations](#monitoring--operations)

## Project Overview

A robust, scalable, and efficient backend for analyzing social media data, built with Go and Firebase Firestore. This platform is designed to handle high write loads while ensuring data consistency and reliability across a distributed architecture.

## Core Architecture & Technologies

This project follows **Clean Architecture (N-Tier)** principles to decouple business logic from the database provider:
-   **Models:** Pure data structures with `json` and `firestore` tags.
-   **Repositories:** Dedicated layer for Firestore SDK operations (Queries, Selects, and CRUD).
-   **Language & Framework:** **Go** with the **Gin** web framework for building high-performance REST APIs.
-   **Services:** The "Brain" of the app. Handles Bcrypt hashing, JWT logic coordination, and Bloom Filter checks.
-   **Controllers:** The core business logic of the application.
-   **Middleware:** A custom JWT interceptor that verifies tokens and injects the userID into the request context.

## Key Design Concepts

DataTracker leverages distributed systems concepts to balance performance and cost-efficiency.

### Replication Strategies

While Firestore handles the physical replication, our Go backend is architected to respect these models:

-   **Multi-Leader Replication:** By using Firestore in Native Mode, data is automatically replicated across multiple geographic regions to survive regional outages.
-   **Leaderless Replication:** Our Event Ingestion is designed to be idempotent. Using Firestore's Set operations allows us to treat writes as leaderless-style updates where the "latest" timestamp wins.
-   **Single-Leader Replication:** For critical user operations (like account creation), we utilize Firestore’s primary leader for Strong Consistency to ensure two people can't claim the same username at the exact same millisecond.

### Consistency Models

-   **Consistent Prefix Reads:** Implemented for user profiles. When a user updates their password or name, the Service Layer ensures the next GET request retrieves the updated document directly from the primary shard.
-   **Eventual Consistency:** Used for analytical counters and public social feeds, allowing the system to scale globally without waiting for every secondary node to sync.

### Advanced Data Structures

-   **Dual Bloom Filters:** To minimize Firestore "Egress" costs and "Read" quotas, we use a Concurrent Bloom Filter as a Read-Optimized Negative Cache.
    -   **logic:** Before querying Firestore to see if an email/username is taken, the system checks a bitset in RAM (O(1) time).
    -   **warm-up:** On server startup, the system performs a "Projection Query" to populate the filter with existing identifiers.

---

## Installation and Setup

Follow these steps to get the DataTracker backend running on your local machine.

### Prerequisites

- [Go](https://go.dev/doc/install) (version 1.18 or newer)
- A Google Cloud Project with **Firestore** enabled.
- [Postman](https://www.postman.com/downloads/) for API testing.

### 1. Google Cloud & Firestore Initialization
Before the code can connect, you must manually prepare the environment:
1.  **Enable Firestore API:** Visit the [Google API Console](https://console.cloud.google.com/apis/api/firestore.googleapis.com/) and enable the API for your project.
2.  **Create Database:** Go to the [Firestore Tab](https://console.cloud.google.com/datastore/setup), select **Native Mode**, and choose a geographic region near you.
3.  **Generate Service Account:** - Navigate to **Project Settings > Service Accounts**.
    - Click **Generate New Private Key** (JSON).
    - Rename this file to `firebaseServiceAccount.json` and place it in the `config/` directory.

### 2. Environment Configuration
Create a file named `.env` inside the `config/` folder and add your JWT secret:
```env
JWT_SECRET_KEY=your_random_secure_string_here
GOOGLE_APPLICATION_CREDENTIALS=config/firebaseServiceAccount.json
```

### 3. Install Dependencies

The project uses Go Modules to manage dependencies. Run the following commands to download and install the required packages:

```bash
go get github.com/gin-gonic/gin
go get firebase.google.com/go/v4
go get cloud.google.com/go/firestore
go get golang.org/x/crypto/bcrypt
go get github.com/joho/godotenv
go get github.com/golang-jwt/jwt/v5
go get github.com/gin-contrib/cors
go mod tidy
```

---

## Running the Application

Once the setup is complete, you can run the server:

```bash
go run main.go
```

The server should start, and you will see output from the Gin framework indicating it is listening for requests on `localhost:8081`. Watch the logs to confirm the Bloom Filter successfully loads existing usernames and emails from Firestore.
