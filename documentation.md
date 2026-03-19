# DataTracker and InkToChat Documentation

A robust, scalable, and efficient backend for analyzing social media data, built with Go and Firebase Firestore. This platform's objectibe is to be a design capable of handling high write loads while ensuring data consistency and reliability across a distributed architecture.

## Table of Contents

- [Project Overview](#project-overview)
- [Core Architecture & Technologies](#core-architecture--technologies)
- [Key Design Concepts](#key-design-concepts)
  - [Replication Strategies](#replication-strategies)
  - [Consistency Models](#consistency-models)
  - [Advanced Data Structures](#advanced-data-structures)


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

