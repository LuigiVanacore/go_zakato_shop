# Parte 4 — Postgres via Docker Compose + prima tabella `users`

> [← Indice della guida](README.md) · [← Parte 3](03-containerizzazione.md)

### Obiettivo
Far partire Postgres accanto ad `auth-service` con un solo comando, creare la prima tabella
`users`, e far sì che `auth-service` si connetta davvero al database all'avvio. Alla fine di
questa parte l'infrastruttura è pronta: nella Parte 5 scriveremo la vera logica di
registrazione sopra a questa base.

### Concetti che userai in questo passo
- **Docker Compose**: un file YAML che descrive **più** container che devono lavorare
  insieme (qui: `postgres` e `auth-service`). Un solo comando li fa partire tutti, in una
  rete privata condivisa dove ogni container raggiunge gli altri **per nome del servizio**
  (es. `auth-service` si connette a `postgres`, non a `localhost`)
- **Postgres**: il database relazionale che useremo per questo servizio (ricorda: database
  per servizio, come deciso nel piano generale)
- **Variabili d'ambiente + file `.env`**: le credenziali del database non vanno scritte a
  mano nei file versionati — le mettiamo in un file `.env` locale, ignorato da Git
- **Driver Postgres in Go**: `auth-service` userà `database/sql` (libreria standard, generica
  per qualsiasi database) insieme a un **driver** specifico per Postgres. È la prima vera
  dipendenza esterna del progetto: da qui in poi vedrai comparire `go.sum`
- **Script di init vs migrazioni vere**: per creare la tabella useremo per ora lo script di
  inizializzazione automatica di Postgres (semplice, ma gira **solo** la primissima volta che
  il database viene creato). Uno strumento di migrazioni vero e proprio (`golang-migrate`,
  già previsto nel piano generale) arriverà quando lo schema inizierà a cambiare nel tempo —
  non ci serve ancora, sarebbe complessità aggiunta troppo presto

---

### Passo 1 — Crea `deploy/.env.example`
Crea il file `deploy/.env.example` (questo file **va versionato**, è solo un modello):
```
POSTGRES_USER=zakato
POSTGRES_PASSWORD=changeme
POSTGRES_DB=zakato_shop
```

### Passo 2 — Crea il tuo `deploy/.env` reale
Copia `deploy/.env.example` in `deploy/.env` (stesso contenuto va benissimo, sei in locale) e
aggiungi al file `.gitignore` nella root del repo questa riga:
```
deploy/.env
```
Perché due file: `.env.example` mostra a chiunque (anche a te tra sei mesi) quali variabili
servono, senza esporre credenziali reali. `.env` è quello che Docker Compose legge davvero, e
non finisce mai su Git — anche se qui sono credenziali finte di sviluppo, è l'abitudine giusta
da avere fin da subito.

### Passo 3 — Script SQL per la tabella `users`
Crea `deploy/init/001_create_users.sql`:
```sql
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```
`gen_random_uuid()` è disponibile in Postgres senza bisogno di estensioni a partire dalla
versione 13 (noi useremo l'immagine `postgres:16`). Usiamo un `UUID` invece di un intero
auto-incrementale come chiave primaria: in un sistema a microservizi è una pratica comune,
perché un ID generato così non rischia mai di collidere tra database diversi.

### Passo 4 — `docker-compose.yml`
Crea `deploy/docker-compose.yml`:
```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./init:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: 5s
      timeout: 5s
      retries: 5

  auth-service:
    build:
      context: ../backend
      dockerfile: services/auth-service/Dockerfile
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  pgdata:
```

Cosa fa, a blocchi:
- `image: postgres:16`: invece di un `Dockerfile`, usiamo direttamente l'immagine ufficiale
  di Postgres pubblicata su Docker Hub — non dobbiamo costruircela noi
- Le tre `POSTGRES_*` vengono lette dal tuo `deploy/.env` (Compose lo carica automaticamente
  se si trova nella stessa cartella del `docker-compose.yml`)
- `volumes: - pgdata:/var/lib/postgresql/data`: i dati del database vengono salvati in un
  "volume" gestito da Docker, così **sopravvivono** anche se fermi e ricrei il container
  (altrimenti ogni riavvio perderesti tutto)
- `volumes: - ./init:/docker-entrypoint-initdb.d`: questo è un trucco specifico
  dell'immagine ufficiale di Postgres — **tutti** gli script `.sql` (o `.sh`) in questa
  cartella vengono eseguiti automaticamente, in ordine alfabetico, **ma solo la primissima
  volta** che il container parte con un volume dati vuoto. Per questo il file si chiama
  `001_create_users.sql`: il prefisso numerico garantisce l'ordine quando in futuro ne
  aggiungeremo altri
- `healthcheck`: Docker esegue periodicamente `pg_isready` dentro il container per sapere se
  Postgres è "pronto ad accettare connessioni", non solo "avviato" (Postgres impiega qualche
  istante dopo l'avvio prima di essere davvero pronto)
- `depends_on: postgres: condition: service_healthy`: dice a Compose "avvia `auth-service`
  solo **dopo** che l'healthcheck di `postgres` è verde" — senza questo, `auth-service`
  potrebbe partire e provare a connettersi prima che Postgres sia pronto, fallendo
- `build: context: ../backend`: il contesto di build (vedi Parte 3) è relativo a **questo**
  file, quindi `../backend` punta correttamente a dove sta `go.mod`

### Passo 5 — Aggiorna il `Dockerfile`: aggiungi `go.sum`
Nella Parte 3, `go.sum` non esisteva ancora (nessuna dipendenza esterna). Ora che ne stiamo
per aggiungere una (Passo 6), aggiorna `backend/services/auth-service/Dockerfile`:
```dockerfile
COPY go.mod go.sum ./
```
(prima era solo `COPY go.mod ./`). `go.sum` contiene i checksum delle dipendenze: garantisce
che ogni build usi esattamente lo stesso codice, byte per byte, anche in futuro.

### Passo 6 — Aggiungi il driver Postgres al modulo Go
Da `backend/`:
```powershell
cd backend
go get github.com/jackc/pgx/v5/stdlib
```
Questo comando aggiorna `go.mod` (aggiunge la dipendenza) e crea/aggiorna `go.sum`. `pgx` è il
driver Postgres più usato nell'ecosistema Go moderno; lo useremo tramite `stdlib`, un
adattatore che lo rende compatibile con il pacchetto standard `database/sql`.

### Passo 7 — Connetti `auth-service` al database
In `backend/services/auth-service/cmd/server/main.go`, aggiungi gli import necessari:
```go
import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)
```
(l'`_` davanti all'import di `pgx/v5/stdlib` significa "importalo solo per il suo effetto
collaterale" — qui, registrarsi come driver disponibile per `database/sql` — senza usare
direttamente nessun suo simbolo)

E, **all'inizio** della funzione `main()`, prima di creare il router:
```go
dbURL := os.Getenv("DATABASE_URL")
if dbURL == "" {
	log.Fatal("DATABASE_URL non impostata")
}

db, err := sql.Open("pgx", dbURL)
if err != nil {
	log.Fatalf("impossibile aprire la connessione al database: %v", err)
}
defer db.Close()

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
if err := db.PingContext(ctx); err != nil {
	log.Fatalf("impossibile raggiungere il database: %v", err)
}
log.Println("connesso al database")
```

Cosa fa:
- `os.Getenv("DATABASE_URL")`: legge la variabile d'ambiente impostata da Compose (Passo 4)
- `sql.Open("pgx", dbURL)`: **non** apre subito una connessione di rete — prepara solo
  l'oggetto `*sql.DB` (che internamente gestisce un pool di connessioni)
- `db.PingContext(ctx)`: qui avviene davvero il primo contatto col database. Se fallisce,
  usciamo subito con `log.Fatalf` — meglio far fallire il servizio **all'avvio**, con un
  errore chiaro nei log, piuttosto che farlo partire "rotto" e scoprirlo alla prima richiesta
- `context.WithTimeout(..., 5*time.Second)`: se il database non risponde entro 5 secondi,
  rinunciamo invece di restare bloccati per sempre

### Passo 8 — Avvia tutto con Compose
```powershell
cd deploy
docker compose up --build
```
`--build` forza la ricostruzione dell'immagine di `auth-service` (utile ogni volta che cambi
il codice Go). Nei log dovresti vedere prima Postgres diventare pronto, poi:
```
auth-service-1  | 2026/... connesso al database
auth-service-1  | 2026/... auth-service in ascolto su :8080
```

### Passo 9 — Verifica
In un altro terminale:
```powershell
curl.exe http://localhost:8080/health
```
Deve rispondere `OK` come prima — la parte nuova è che ora, se il database **non** fosse
raggiungibile, `auth-service` non sarebbe nemmeno partito (vedi Passo 7).

Controlla anche che la tabella esista davvero:
```powershell
docker compose exec postgres psql -U zakato -d zakato_shop -c "\dt"
```
Dovresti vedere `users` nell'elenco delle tabelle.

### Nota: perché `/health` non controlla il database
Potresti pensare "perché non faccio rispondere `/health` con errore se il database è giù?".
In Kubernetes si distinguono due controlli diversi: la **liveness** ("il processo è vivo? se
no, riavvialo") e la **readiness** ("è pronto a ricevere traffico? se no, non mandargliene,
ma non serve riavviarlo"). Se `/health` (liveness) dipendesse dal database, un problema
temporaneo di Postgres farebbe riavviare in loop anche `auth-service`, anche se il processo
Go è perfettamente sano — inutile e controproducente. Aggiungeremo un endpoint `/ready`
dedicato quando arriveremo a configurare le probe di Kubernetes.

### Problemi comuni
- **`connection refused` all'avvio di `auth-service`**: quasi sempre l'healthcheck/`depends_on`
  manca o non ha ancora avuto tempo di diventare verde — verifica con `docker compose ps` che
  `postgres` sia `healthy`, non solo `running`
- **La tabella `users` non c'è**: lo script di init gira solo su volume dati vuoto. Se hai già
  avviato Postgres prima di creare `001_create_users.sql`, il volume esiste già e lo script
  non verrà rieseguito. Soluzione: `docker compose down -v` (rimuove anche i volumi) e poi
  `docker compose up --build` da capo
- **Porta 5432 già in uso**: hai già un Postgres installato/in esecuzione sul PC. Cambia il
  mapping in `docker-compose.yml`, es. `"5433:5432"` (la porta a sinistra è quella sul tuo PC)
- **`password authentication failed`**: `deploy/.env` non è allineato a quanto Postgres ha
  effettivamente memorizzato al primo avvio — se cambi la password dopo che il volume esiste
  già, serve `docker compose down -v` per farla ripartire da zero

### Checklist di fine parte
- [ ] `docker compose up --build` (da `deploy/`) avvia sia `postgres` che `auth-service` senza errori
- [ ] I log di `auth-service` mostrano `connesso al database`
- [ ] `docker compose exec postgres psql -U zakato -d zakato_shop -c "\dt"` mostra la tabella `users`
- [ ] `curl.exe http://localhost:8080/health` risponde ancora `OK`
- [ ] Hai capito la differenza tra liveness e readiness, e perché per ora non le confondiamo

Quando questa checklist è verde, siamo pronti per la **Parte 5** (endpoint di registrazione:
hashing della password, salvataggio del primo utente reale) — la scrivo in questa cartella
quando mi dici che sei arrivato fin qui.

> [← Indice della guida](README.md) · [← Parte 3](03-containerizzazione.md)
