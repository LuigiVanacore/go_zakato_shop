# Parte 3 — Containerizzazione con Docker

> [← Indice della guida](README.md) · [← Parte 2](02-test-automatico.md)

### Obiettivo
Far girare `auth-service` dentro un container Docker, invece che con `go run` sulla tua
macchina. È il primo passo verso Kubernetes: se non gira in un container, non può girare in
un cluster.

### Concetti che userai in questo passo
- **Immagine**: un pacchetto "congelato" che contiene il programma già compilato più tutto
  ciò che gli serve per girare (in Go, quasi nulla: un binario statico)
- **Container**: un'istanza *in esecuzione* di un'immagine — come un processo, ma isolato
- **`Dockerfile`**: la ricetta che descrive come costruire un'immagine, passo per passo
- **Build multi-stage**: usiamo **due** immagini in sequenza — una con il compilatore Go
  per costruire il binario, e una finale minimale che contiene *solo* il binario compilato.
  Risultato: l'immagine che finisce in produzione è piccola (pochi MB) e non contiene
  compilatore, codice sorgente o strumenti extra che un attaccante potrebbe sfruttare

### Passo 0 — Verifica di avere Docker
```powershell
docker version
```
Deve rispondere con informazioni sia di Client che di Server. Se il Server non risponde
("Cannot connect to the Docker daemon..."), apri Docker Desktop e aspetta che finisca
l'avvio prima di continuare.

### Passo 1 — Scrivi il `Dockerfile`
Crea `backend/services/auth-service/Dockerfile`:

```dockerfile
# Stage 1: build — usa l'immagine ufficiale Go, con compilatore e toolchain
FROM golang:1.26 AS builder
WORKDIR /src

# Copiamo prima SOLO i file delle dipendenze: se il codice sorgente cambia ma go.mod no,
# Docker riusa la cache di questo layer invece di riscaricare le dipendenze ogni volta
COPY go.mod ./
RUN go mod download

# Ora copiamo tutto il resto del modulo (tutti i servizi, dato che il modulo Go è unico)
COPY . .

# CGO_ENABLED=0 produce un binario statico, senza dipendenze dalla libc di sistema —
# necessario per poterlo far girare sull'immagine finale "distroless" che non ha nulla
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auth-service ./services/auth-service/cmd/server

# Stage 2: immagine finale — minimale, niente compilatore, niente shell
FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/auth-service /auth-service
EXPOSE 8080
ENTRYPOINT ["/auth-service"]
```

Cosa fa, riga per riga:
- `FROM golang:1.26 AS builder`: parte da un'immagine con Go già installato, e le dà il nome
  `builder` (per poterla richiamare dopo)
- `WORKDIR /src`: da qui in poi i comandi girano dentro `/src` **nel container**, non sul tuo PC
- `COPY go.mod ./` + `RUN go mod download`: scarica le dipendenze del modulo (per ora nessuna,
  usiamo solo la libreria standard — ma tra poco ne avremo, es. per Postgres e JWT)
- `COPY . .`: copia tutto il contenuto della cartella di build (vedi Passo 2) dentro `/src`
- `RUN ... go build -o /out/auth-service ./services/auth-service/cmd/server`: compila
  **solo** il pacchetto `main` di `auth-service`, producendo un unico eseguibile
- `FROM gcr.io/distroless/static-debian12`: la seconda immagine, quella che verrà davvero
  distribuita — "distroless" significa "senza distribuzione Linux completa", solo il minimo
  indispensabile per eseguire un binario statico
- `COPY --from=builder /out/auth-service /auth-service`: prende **solo il binario compilato**
  dallo stage precedente, scartando compilatore e codice sorgente
- `EXPOSE 8080`: documentazione (non è strettamente necessaria a runtime) che dice "questo
  container ascolta sulla porta 8080"
- `ENTRYPOINT ["/auth-service"]`: il comando eseguito quando il container parte

### Passo 2 — Build dell'immagine
Il `Dockerfile` fa riferimento a `go.mod` e a tutti i servizi con percorsi relativi (es.
`./services/auth-service/...`): per questo la build va lanciata da dentro `backend/`, che è
la cartella che contiene `go.mod` (il "contesto" della build, cioè l'insieme di file visibili
al `Dockerfile`):

```powershell
cd backend
docker build -t auth-service:dev -f services/auth-service/Dockerfile .
```
- `-t auth-service:dev`: dà un nome (tag) all'immagine costruita
- `-f services/auth-service/Dockerfile`: indica dove si trova il Dockerfile (non è nella
  cartella corrente)
- `.` (il punto finale): il contesto di build è la cartella corrente (`backend/`)

La prima build richiede un po' di tempo (scarica le immagini base); le successive sono più
veloci grazie alla cache dei layer.

### Passo 3 — Avvia il container
```powershell
docker run --rm -p 8080:8080 --name auth-service auth-service:dev
```
- `-p 8080:8080`: collega la porta 8080 del tuo PC alla porta 8080 dentro il container
  (formato `host:container`)
- `--rm`: rimuove automaticamente il container quando si ferma (utile in sviluppo, per non
  accumulare container morti)
- `--name auth-service`: dà un nome al container, comodo per riferirtici dopo

Il terminale resta "agganciato" ai log del container (li vedi scorrere in tempo reale).

### Passo 4 — Verifica che funzioni
Come nella Parte 1, ma ora la richiesta arriva a un processo che gira **dentro Docker**:
```powershell
curl.exe http://localhost:8080/health
```
Ti aspetti di nuovo `OK` con status `200`. Se funziona, hai appena dimostrato che il
container è completamente autosufficiente — non dipende dal tuo `go run` locale.

Per fermare: `Ctrl+C` nel terminale dove gira, oppure da un altro terminale
`docker stop auth-service`.

### Passo 5 (facoltativo) — Farlo girare in background
Se non vuoi tenere un terminale occupato:
```powershell
docker run -d --rm -p 8080:8080 --name auth-service auth-service:dev
docker logs -f auth-service   # per vedere i log quando vuoi
docker stop auth-service      # per fermarlo
```
`-d` = "detached", il container parte in background e ti restituisce subito il prompt.

### Problemi comuni
- **`Cannot connect to the Docker daemon`**: Docker Desktop non è avviato — aprilo e aspetta
- **`failed to compute cache key` / file non trovato durante `COPY`**: quasi sempre significa
  che hai lanciato `docker build` dalla cartella sbagliata (deve essere `backend/`, non la
  root del repo e non `services/auth-service/`)
- **Port già in uso**: stesso discorso della Parte 1, ma qui è la porta *host* (la prima nel
  `-p 8080:8080`) — cambiala, es. `-p 8081:8080`
- **Vuoi "entrare" nel container per debug**: con `distroless` non puoi (non ha una shell,
  è voluto, per sicurezza). Se ti serve investigare un problema, ricostruisci temporaneamente
  lo stage finale da `golang:1.26` invece che da `distroless` — poi torna a `distroless`

### Checklist di fine parte
- [ ] `docker build` completa senza errori e produce `auth-service:dev`
- [ ] `docker run` avvia il container e `curl.exe http://localhost:8080/health` risponde `OK`
- [ ] Hai capito perché usiamo due stage (build vs immagine finale) e cosa contiene ciascuno
- [ ] Hai capito perché la build va lanciata da `backend/` e non da altre cartelle

Quando questa checklist è verde, siamo pronti per la **Parte 4** (Postgres via Docker Compose
e prima tabella `users`) — la scrivo in questa cartella quando mi dici che sei arrivato fin qui.

> [← Indice della guida](README.md) · [← Parte 2](02-test-automatico.md)
