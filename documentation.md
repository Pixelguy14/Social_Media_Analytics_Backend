# Backend Engineering Documentation: DataTracker & InkToChat

This document provides a comprehensive overview of the architectural decisions, security measures, and data optimizations implemented in the DataTracker and InkToChat distributed backend.

---

## 1. Core Architecture & Technologies

### The Tech Stack
- **Primary Language:** [Go](https://go.dev/) (v1.25+), chosen for its performance, concurrency primitives (goroutines), and robust standard library.
- **Web Framework:** [Gin Gonic](https://gin-gonic.com/), a high-performance HTTP web framework.
- **Database:** [Google Cloud Firestore](https://firebase.google.com/docs/firestore), used in **Native Mode** for document-oriented ACID transactions and global scalability.
- **Authentication:** [Firebase Auth](https://firebase.google.com/docs/auth) for custom token generation and [JWT](https://jwt.io/) for standard session management.
- **Probabilistic Data Structures:** `bits-and-blooms/bloom` for high-speed existence checks.
- **Email Systems:** [net/smtp](https://pkg.go.dev/net/smtp) for secure, credential-backed delivery via OCI (Oracle Cloud Infrastructure).

### Clean Architecture (N-Tier)
The project follows a strict separation of concerns to ensure maintainability and testability:
1.  **Models Layer:** Defines the core data structures with both `json` (API) and `firestore` (DB) tags.
2.  **Repository Layer:** The only layer permitted to touch the Firestore SDK. It abstracts data persistence logic behind interfaces.
3.  **Service Layer:** The "Brain" of the application. Orchestrates complex business rules (e.g., admin color verification, analytics tracking, password hashing logic).
4.  **Controller Layer:** Entry points for HTTP requests. Handles request binding, rate limiting, and mapping service responses to JSON outputs.
5.  **Middleware Layer:** Global/Group-level interceptors for security and context injection.

---

## 2. Infrastructure & Distributed Design Concepts

### Replication & Consistency Models
- **Leaderless Writes:** Event ingestion is idempotent using Firestore's `Set` operations, allowing for resilient, eventually consistent updates.
- **Strong Consistency:** Applied to critical operations like user registration to prevent race conditions during account creation.
- **Global Distribution:** Leveraging Firestore Native Mode for automatic multi-regional replication with 99.999% availability.

### Probabilistic Optimizations
- **Dual Concurrent Bloom Filters:** Implemented as a "Read-Optimized Negative Cache" to prevent unnecessary Firestore lookups during username/email availability checks.
- **Background Refresh Loop:** A dedicated goroutine rebuilds the filters every 24 hours to clear "false positive" saturation and maintain accuracy over long server uptimes.

---

## 3. Data-Intensive Code & Optimizations

### High-Write Messaging Architecture
- **Circular Buffer Workers:** To keep database costs low and performance high, background workers monitor chat lobbies (A, B, C, D). They automatically prune collections to maintain a **100-message buffer** per room using descending timestamp offsets.
- **Asynchronous Analytics:** Engagement tracking is offloaded to background goroutines, ensuring that tracking metrics (messages sent, drawings created) doesn't block the critical path of responding to the user.

### Intelligent Data Formats
- **1-Bit Map Validation:** InkToChat utilizes a recursive bit-array algorithm to validate drawing blobs. This ensures that users cannot submit bloated or malformed binary data to the lobbies.
- **Token Bucket Rate Limiting:** A custom `ratelimit` manager uses a moving-average algorithm to prevent spam on sensitive endpoints (Token generation, Messaging, Drawing) while allowing for natural bursts of user interaction.

---

## 4. Security Measures & Auth Infrastructure

- **Bcrypt Password Hashing:** All user passwords are encrypted using Bcrypt with a **Cost Factor of 12**, exceeding modern OWASP requirements for brute-force resistance.
- **JWT Hardening:** 
    - **Expiration:** Tokens are strictly set to **24 hours**.
    - **Algorithm Pinning:** The middleware enforces HMAC-SHA256, preventing "Algorithm Confusion" attacks (e.g., `alg: none` exploits).
    - **Dependency Injection:** The `JWT_SECRET_KEY` is validated and injected into closures at startup, failing fast if the environment is misconfigured.
- **Middleware Guard Rails:**
    - **`AdminOnly`:** Protects administrative analytics and destructive system resets.
    - **`OwnerOrAdmin`:** An ownership middleware that ensures users can only read/edit/delete their own data while still allowing administrative overrides.
- **PII Governance:** Debug logging is gated behind `GIN_MODE` checks to ensure user emails/IDs do not leak into production log management systems.
- **Internationalized Password Security:** Our validator uses `unicode.IsUpper` to support global uppercase characters (like **Ñ, Á, Ç**) while enforcing a strict 8+ character minimum with mixed numbers/letters.
- **Secure Recovery & Email (Zero-Knowledge Reset):** 
    - **Raw Token Entropy:** Reset tokens are 32-byte high-entropy strings generated via `crypto/rand`.
    - **At-Rest Hashing:** Tokens are never stored in the database in plain text; only their **SHA-256 hashes** are persisted.
    - **One-Time Use Enforcement:** The backend implements a mandatory "Verification Deletion" logic where tokens are destroyed immediately upon retrieval from Firestore, preventing replay attacks.
    - **Infrastructure Orchestration:** Detailed setup for SPF, DKIM, and DMARC is documented in [Infrastructure Mail Setup](file:///home/pixel/Documents/2026/Github_Portfolio/distributed_systems/social_media_analytics_backend/mail_setup.md).
- **Input Redaction:** Passwords and sensitive tokens utilize `json:"-"` tags to ensure they are never serialized in API responses.
- **RS256 Asymmetric encription:** In asymmetric cryptography, we split the "secret" into two distinct parts: private.pem (only the server has access to this) and public.pem (this is what we give to the client). 

    Previously, we used HS256, which uses a single JWT_SECRET_KEY for both signing and verifying. If that one secret was leaked, an attacker could forge any token they wanted.

    With RS256, even if your public key were leaked, your system remains secure because the private key (the only thing capable of forging tokens) stays locked away on the server

```bash
# Generate RS256 Asymmetric Key Pair
openssl genrsa -out config/private.pem 2048 && openssl rsa -in config/private.pem -pubout -out config/public.pem
```

- **Sensitive Rate Limiting:** While standard endpoints use a burst-friendly Token Bucket, the **Password Reset** flow is protected by a dedicated `ResetRateLimiter`. This restricts attempts to **3 per hour per IP**, mitigating both email spam and brute-force enumeration.


---

## 5. Firebase Design Principles

### Data Modeling Patterns
- **Sub-collections:** Hierarchical modeling (e.g., `rooms/{id}/messages`) allows for isolated indexing and efficient pruning without affecting the performance of other chat lobbies.
- **Indempotent Set Operations:** Using `MergeAll` patterns to ensure analytical counters can survive duplicate transmission in distributed environments.
- **Server Timestamps:** Utilizing `firestore.ServerTimestamp` to ensure perfectly synchronized linear order across distributed clients, regardless of local machine clock drift.
