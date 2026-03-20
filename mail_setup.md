# Infrastructure Orchestration: SMTP & Email Deliverability

This document outlines the multi-phase setup for integrating **Oracle Cloud Infrastructure (OCI)** with **Cloudflare DNS** to ensure 100% email deliverability and high sender reputation (reaching a 10.0/10 [Mail-Tester](https://www.mail-tester.com) score).

---

## Phase 1: The Domain Foundation (DigitalPlat)
**Goal:** Delegate domain management to a higher-level network layer.

1.  **Nameserver Handover:** Locate the Nameserver settings for your domain in the DigitalPlat dashboard.
2.  **Clean Slate:** Remove any default NS records provided by the register.
3.  **The Cloudflare Link:** Replace them with the specific nameservers provided by your Cloudflare account.
4.  **Security Lock:** Ensure "Register Lock" is enabled to prevent unauthorized domain hijacking.
    - *Rationale:* Decoupling DNS management from the register provides a layer of protection against social engineering attacks targeting the domain provider.

---

## Phase 2: The Network Guardian (Cloudflare)
**Goal:** Route traffic, enforce SSL/TLS, and establish domain authority for emails.

### 1. Networking (A Records & Proxying)
- **A Record:** Path: `@`, Value: `[Oracle_IP_Address]`, Proxy: **Enabled** (Orange Cloud).
- **Security Rationale:** Cloudflare's Reverse Proxy hides your real Oracle IP from the public, mitigating direct DDoS attacks on your instance.

### 2. Mandatory SSL/TLS Settings
- **SSL Mode:** Full (Strict).
- **Minimum TLS:** 1.2.
- **Always Use HTTPS:** Enabled.

### 3. Email Authentication (The "Trust" Stack)
Add these records to Cloudflare to prove your server's identity to Gmail, Outlook, etc.

| Type | Name | Content | Security Rationale |
| :--- | :--- | :--- | :--- |
| **TXT (SPF)** | `@` | `v=spf1 include:rp.oracleemaildelivery.com ~all` | Authorizes OCI to send email on behalf of your domain. |
| **CNAME (DKIM)** | `[selector]._domainkey` | `[Provided_by_Oracle]` | Signs outgoing emails with a cryptographic signature. |
| **TXT (DMARC)** | `_dmarc` | `v=DMARC1; p=none;` | Tells receiving servers how to handle emails that fail SPF/DKIM. |

---

## Phase 3: The SMTP Foundry (Oracle Cloud)
**Goal:** Open network ports and configure the authenticated delivery engine.

### 1. VCN Ingress Rules (The Gates)
Navigate to your Virtual Cloud Network (VCN) and add the following Ingress Rules to your Security List:
- **Port 80 (HTTP):** Required for initial ACME certificate challenges.
- **Port 443 (HTTPS):** Critical for secure user traffic.
- **Port 587 (SMTP):** Oracle's standard port for authenticated STARTTLS email delivery.

### 2. Email Delivery Configuration
- **Approved Senders:** You must manually add `noreply@yourdomain.com` here. OCI will reject any email attempt from an unlisted address.
- **DKIM Generation:** Create a DKIM record in OCI first to get the selector and target values for the Cloudflare CNAME.
- **SMTP Credentials:** Generate a dedicated credential pair. These will be used for `SMTP_USER` and `SMTP_PASS` in your `.env`.

---

## Phase 4: Final Handshake & Verification
Before going live, verify the "Trust Chain" is complete:

1.  **DKIM Activity:** Status in OCI Console must be **Active**.
2.  **SPF Alignment:** Approved Senders status must be **Configured**.
3.  **Proxy Check:** `ping yourdomain.com` must resolve to a Cloudflare IP (e.g., `104.x.x.x`), proving your real IP is masked.
4.  **Port Scan:** Use `nmap` or an online checker to verify ports 80 and 443 are **Open**.
5.  **Score Check:** Use [Mail-Tester](https://www.mail-tester.com) to target a 10/10 score. Any failure in DKIM/SPF will result in your emails landing in the Spam folder.