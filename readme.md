# DataTracker: Social Media Analytics Backend & InkToChat: Pictochat Replication (V3)

**InkToChat** is a modern, distributed replication of the classic Nintendo DS Pictochat experience. It leverages a **Go-based orchestration layer** for business logic and validation, combined with **Firebase (Firestore & Realtime DB)** for global state management and real-time synchronization.

## Project Objective
Replicate real-time text and image communication across four persistent lobbies (A, B, C, & D) while enforcing hardware-inspired constraints:
- **100-Message Circular Buffer:** A Go worker automatically prunes the oldest message when a room exceeds 100 entries.
- **1-Bit Drawing Validation:** Strictly validates binary-encoded drawings (256x192 px, Black/White) to ensure payload integrity (exactly 6,144 bytes per drawing).
- **Zero-Friction Auth:** Anonymous username reservation backed by a **Bloom Filter** to minimize database lookups.
- **Global Localization:** Support for 10 languages (EN, DE, FR, ES, IT, JP, HU, FI, PT, NL).

## Technical Architecture
InkToChat follows a **Hybrid Cloud Architecture**, separating managed synchronization from custom business rules.

### Go Orchestration Layer (The Gatekeeper)
- **Auth (Bloom Filter):** Uses a read-optimized negative cache in RAM to check username availability before hitting Firestore.
- **Worker (Circular Buffer):** A background Go routine monitors Firestore collections and enforces the 100-message limit per room.
- **Validation (Bit-Array):** Ensures drawing blobs match the 6,144-byte constraint before they are written to the database.
- **Rate Limiting:** Implements a **Token Bucket** algorithm to protect Firebase free-tier quotas.
- **i18n:** Server-side localization for system messages using `nicksnyder/go-i18n`.

- **Security Rules:** Prevents users from spoofing others' usernames.

## System Integration (The Binding)
InkToChat is not just a standalone toy; it is **deeply integrated** with the DataTracker analytics engine to prove the scalability of a unified architecture:
- **Unified Identity:** The Go Gatekeeper automatically identifies if a lobby user exists in the persistent `DataTracker` database. If they do, their session is linked to their permanent profile ID.
- **Engagement Engine:** Every drawing and message is asynchronously tracked by a custom **Analytics Service**. This transforms volatile chat activity into persistent business metrics (e.g., "Hourly Engagement Rate," "User Creativity Index").
- **Shared Infrastructure:** Both systems share the same Firestore client, unified logging (`slog`), and a centralized rate-limiting manager, demonstrating professional resource reuse.

## Installation & Setup

### Prerequisites
- [Go](https://go.dev/doc/install) (v1.25+)
- [Firebase CLI](https://firebase.google.com/docs/cli)
- A Google Cloud Project with Firestore and Auth enabled.


### 1. Google Cloud & Firestore Initialization

Before the code can connect, you must manually prepare the environment:

1. **Enable Firestore API:** Visit the [Google API Console](https://console.cloud.google.com/apis/api/firestore.googleapis.com/) and enable the API for your project.

2. **Create Database:** Go to the [Firestore Tab](https://console.cloud.google.com/datastore/setup), select **Native Mode**, and choose a geographic region near you.

3. **Generate Service Account:** - Navigate to **Project Settings > Service Accounts**.

- Click **Generate New Private Key** (JSON).

- Rename this file to `firebaseServiceAccount.json` and place it in the `config/` directory.


### 2. Environment Configuration

Create a file named `.env` inside the `config/` folder and add your JWT secret:

```env

JWT_SECRET_KEY=your_random_secure_string_here

GOOGLE_APPLICATION_CREDENTIALS=config/firebaseServiceAccount.json

``` 

### 1. Configuration
1. Initialize dependencies: `go mod tidy`
2. Set up your `.env` in `config/`

### 2. Running Locally
```bash
# Start the Go Backend (Port 8081)
go run main.go

# (Optional) Verify security rules with emulator
firebase emulators:start
```

## Admin Analytics
The secondary frontend tasks include building an admin panel to monitor:
- **Active Connections:** Snapshot of current presence via RTDB.
- **Buffer Health:** Monitoring Firestore document counts per lobby.
- **Spam Metrics:** Rate-limiter drop events to identify malicious behavior.
- **Namespace Saturation:** Tracking Bloom Filter utilization.
