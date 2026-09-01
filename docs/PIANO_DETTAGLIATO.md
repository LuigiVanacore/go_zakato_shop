# Zakato Shop — Piano Dettagliato (passo-passo)

Approccio: piccoli passi verificabili, uno alla volta. Solo la **Fase 1** è dettagliata qui.
Alla fine di ogni fase, decidiamo insieme il dettaglio della fase successiva e la aggiungiamo
a questo documento — niente pianificazione a lungo raggio fatta in anticipo e poi disattesa.

Riferimento architettura/tecnologie complete: [PIANO_GENERALE.md](PIANO_GENERALE.md)

---

## Fase 1 — Fondamenta locali

**Obiettivo della fase**: avere un singolo microservizio Go minimo, testato, containerizzato —
la base pulita su cui costruire tutto il resto. Niente gRPC, niente DB ancora: solo le fondamenta.

### Step 1 — Decisioni preliminari
Da confermare (o cambiare) prima di scrivere codice:
- Dominio applicativo: e-commerce "Zakato Shop" (proposto, vedi piano generale)
- Struttura repo: **monorepo** (tutti i microservizi + frontend nello stesso repo, cartelle separate).
  Consigliato per un progetto di apprendimento in solitaria: più semplice da gestire rispetto a più repo.
- Versione Go da installare/usare (consigliato: ultima stabile)

### Step 2 — Repository e struttura cartelle
- `git init` nella root del progetto
- Struttura iniziale:
  ```
  /services/auth-service     ← primo microservizio
  /frontend                  ← Angular, arriverà più avanti
  /deploy                    ← docker-compose, k8s, terraform, arriveranno più avanti
  /docs                      ← questo piano e la documentazione
  ```
- `README.md` iniziale con una riga di descrizione del progetto

### Step 3 — Primo microservizio Go minimo: `auth-service`
- Modulo Go **unico per l'intero repo** (mono-modulo): `go mod init` nella **root** del progetto,
  non dentro il singolo servizio. Così tutti i servizi condividono un solo `go.mod`/`go.sum` e possono
  importarsi codice a vicenda (utile più avanti per i pacchetti `.proto` generati e le utility comuni)
  senza `go.work`. Si potrà passare a un modulo per servizio in futuro se servirà davvero.
- Server HTTP minimale (stdlib `net/http`, niente framework per ora) con un solo endpoint:
  `GET /health` → `200 OK`
- Struttura base del progetto Go: `cmd/server/main.go` + `internal/`

### Step 4 — Verifica e primo test
- Avvio locale (`go run`) e verifica manuale con `curl`/browser su `/health`
- Un test automatico Go (`go test`) che chiama l'handler `/health` e verifica lo status code

### Step 5 — Containerizzazione
- `Dockerfile` multi-stage (stage di build + immagine finale minimale, es. `distroless` o `alpine`)
- Build dell'immagine e run locale del container, riverifica `/health` dentro Docker

---

**Fine Fase 1**: a questo punto avremo un servizio Go reale, testato e containerizzato — lo scheletro
che replicheremo per gli altri microservizi. Da lì decidiamo insieme il prossimo blocco (tipicamente:
aggiungere Postgres e le prime query reali all'auth-service, oppure introdurre gRPC) e lo dettagliamo qui.

## Fase 2 — (da dettagliare quando arriviamo qui)

## Fase 3 — (da dettagliare quando arriviamo qui)

...
