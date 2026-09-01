# Zakato Shop — Piano Generale

Progetto di esercitazione: e-commerce a microservizi in **Go + gRPC**, frontend in **Angular**,
containerizzato con **Docker**, orchestrato con **Kubernetes**, infrastruttura **AWS** gestita
con **Terraform**. Obiettivo: imparare bene ogni tecnologia costruendo qualcosa di completo e realistico,
non solo un tutorial giocattolo.

> Questo documento descrive il quadro generale (cosa costruiamo e con cosa).
> Il percorso passo-passo per iniziare è in [PIANO_DETTAGLIATO.md](PIANO_DETTAGLIATO.md).

## 1. Dominio applicativo

Negozio online "Zakato Shop": catalogo prodotti, carrello, checkout, pagamento con carta,
gestione ordini e magazzino, area utente, area admin.

## 2. Funzionalità

- **Utenti**: registrazione, login/logout, refresh token, profilo, reset password
- **Autenticazione/autorizzazione**: JWT (access + refresh), RBAC (customer/admin), OAuth2/OIDC (login Google) come step avanzato
- **Catalogo**: CRUD prodotti (admin), categorie, ricerca, filtri, paginazione
- **Carrello**: add/remove, persistenza (Redis), sincronizzato per utente loggato
- **Ordini**: creazione, stati (pending → paid → shipped → delivered / cancelled), storico
- **Pagamenti**: integrazione Stripe (test mode), mai gestire dati carta lato server, conferma via webhook
- **Magazzino/Inventario**: disponibilità, riserva stock in checkout, rilascio su fallimento/timeout
- **Notifiche**: email di conferma (ordine, registrazione)
- **Admin panel**: gestione prodotti/ordini/utenti
- **Recensioni prodotto** (fase avanzata, opzionale)
- **Ricerca full-text** (fase avanzata, opzionale — Elasticsearch/OpenSearch)
- **Osservabilità**: log strutturati, metriche, tracing distribuito

## 3. Architettura

```
Angular SPA
     │  HTTPS/REST (JSON)
     ▼
API Gateway (Go)  ── REST verso client, gRPC verso i servizi interni
     │  gRPC
     ├── Auth Service
     ├── User Service
     ├── Catalog Service
     ├── Cart Service
     ├── Order Service        ← orchestratore Saga
     ├── Payment Service
     ├── Inventory Service
     └── Notification Service

Message broker (NATS JetStream, poi eventualmente Kafka)
     ↳ eventi di dominio per la Saga + outbox pattern
```

Ogni microservizio ha il proprio database (database-per-service).

## 4. Pattern architetturali

- **Saga (orchestration-based)**: `Order Service` coordina "crea ordine → riserva stock → addebita
  pagamento → conferma → notifica", con compensazioni (rilascia stock, rimborsa) se un passo fallisce
- **Outbox pattern**: consistenza tra scrittura sul DB del servizio e pubblicazione evento sul broker
- **API Gateway pattern**: unico punto di ingresso REST per il frontend
- **Circuit breaker + retry**: resilienza nelle chiamate gRPC tra servizi (es. `sony/gobreaker`)
- **Database-per-service**
- **CQRS leggero sul catalogo** (facoltativo, fase avanzata)

## 5. Stack tecnologico

**Backend**
- Go (ultima stabile), gRPC + Protocol Buffers
- `grpc-gateway` (o gateway custom) per esporre REST/JSON al frontend
- `sqlc` per query type-safe verso Postgres (alternativa più "Go idiomatico" a un ORM)
- `golang-migrate` per le migrazioni
- PostgreSQL (un DB/schema per servizio), Redis (cache/carrello)
- NATS JetStream come message broker iniziale (Kafka come possibile step avanzato)
- `slog` (stdlib) per logging strutturato
- OpenTelemetry per tracing
- `golang-jwt` per JWT, `go-playground/validator` per validazione input

**Frontend**
- Angular (ultima versione, standalone components, Signals per lo state)
- TypeScript, RxJS
- Angular Material o Tailwind CSS
- Stripe.js/Elements per il pagamento

**Sicurezza**
- HTTPS/TLS ovunque (cert-manager in K8s)
- JWT access token (breve durata) + refresh token in cookie httpOnly
- RBAC, rate limiting sull'API Gateway
- Validazione input, query parametrizzate, protezione CSRF/XSS
- Secrets: Docker/K8s Secrets in locale → AWS Secrets Manager in cloud
- Scansione immagini Docker (Trivy) in CI

**DevOps / Infrastruttura**
- Docker (build multi-stage) per ogni servizio
- Docker Compose per lo sviluppo locale
- Kubernetes: `kind`/`minikube` in locale, AWS **EKS** in cloud
- Helm charts per il deployment
- Terraform per l'infrastruttura AWS (VPC, EKS, RDS, ElastiCache, ECR, IAM, Route53, ALB)
- GitHub Actions per CI/CD (build, test, lint, security scan, push ECR, deploy)
- Observability: Prometheus + Grafana, Loki, Jaeger/Tempo (via OpenTelemetry)

**AWS**
- EKS, RDS (Postgres), ElastiCache (Redis), ECR, S3, Route53 + ACM, IAM (IRSA), Secrets Manager, CloudWatch
- (Avanzato/opzionale) MSK se in futuro si migra da NATS a Kafka gestito

## 6. Fasi macro (roadmap, senza dettaglio dei singoli step)

1. Fondamenta locali: un singolo servizio Go + Docker
2. Più microservizi in gRPC, un DB per servizio
3. Autenticazione end-to-end + primo frontend Angular collegato al backend
4. Flusso e-commerce base: catalogo, carrello, ordini (senza pagamenti reali)
5. Saga pattern sul flusso ordine (orchestrazione, compensazioni, outbox, broker)
6. Pagamenti reali con Stripe (test mode)
7. Sicurezza avanzata (rate limiting, RBAC completo, secrets management)
8. Kubernetes locale (kind/minikube) + Helm
9. Observability (log, metriche, tracing)
10. Terraform + deploy su AWS (EKS, RDS, ecc.)
11. CI/CD completo
12. Rifiniture: admin panel, ricerca, recensioni, ottimizzazioni, eventuale service mesh

Ogni fase verrà scomposta in step concreti quando ci arriviamo — vedi il piano dettagliato.
