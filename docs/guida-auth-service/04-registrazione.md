# Parte 4 — Registrazione (`POST /register`) con storage in-memory

> [← Indice della guida](README.md) · [← Parte 3](03-containerizzazione.md)

### Obiettivo
Un endpoint `POST /register` che riceve email e password, valida l'input, e salva l'utente
(con la password **mai in chiaro**, solo il suo hash) in uno storage **in-memory** — una mappa
Go, nascosta dietro un'interfaccia. Alla Parte 7 sostituiremo quello storage con Postgres
**senza toccare** la logica di validazione/hashing che scriviamo qui: è il punto di tutta
questa parte.

### Concetti che userai in questo passo
- **`encoding/json`**: il pacchetto standard per trasformare JSON in struct Go (`Decode`) e
  viceversa (`Encode`). Le annotazioni tipo `` `json:"email"` `` dicono come si chiama il
  campo nel JSON
- **Perché non si salva mai la password in chiaro**: se il database venisse compromesso, un
  attaccante avrebbe direttamente le password di tutti. Si salva invece un **hash**: una
  trasformazione a senso unico (non si può tornare indietro dall'hash alla password)
- **bcrypt**: l'algoritmo di hashing che useremo. A differenza di un hash "semplice"
  (es. SHA-256) è pensato apposta per essere **lento** e include automaticamente un *salt*
  casuale — due utenti con la stessa password ottengono hash diversi, il che rende inutili
  le rainbow table
- **Repository pattern**: definiamo un'**interfaccia** che descrive "cosa si può fare con gli
  utenti" (crearne uno, cercarne uno per email), e per ora scriviamo **una sola
  implementazione** in memoria. Il resto del codice (l'handler HTTP) dipende solo
  dall'interfaccia, mai dalla mappa in memoria direttamente — così alla Parte 7 basterà
  scrivere una nuova implementazione che parla con Postgres
- **`sync.Mutex`**: un server HTTP gestisce richieste **in parallelo** (goroutine diverse per
  richieste diverse). Una mappa Go **non** è sicura se letta e scritta da più goroutine
  contemporaneamente senza sincronizzazione: si rischia una *race condition*, un bug che può
  anche far crashare il programma in modo intermittente. Il mutex garantisce che solo una
  goroutine alla volta acceda alla mappa
- **Due pacchetti sotto `internal/`, non uno**: separiamo **dominio** da **trasporto**:
  - `internal/user`: cos'è un utente e come si salva/recupera — non sa nulla di HTTP
  - `internal/auth`: l'endpoint HTTP di registrazione — sa come si parla con un client via
    JSON, e usa `internal/user` per la parte dati
  Questo si chiama separare per **responsabilità**: se domani cambi il formato della risposta
  HTTP non tocchi `internal/user`; se cambi come sono salvati gli utenti (Parte 7) non tocchi
  `internal/auth`. Ricorda anche che un pacchetto sotto `internal/` può essere importato
  **solo** da codice dentro lo stesso ramo del modulo — è un modo per dire "dettaglio
  implementativo di `auth-service`, non un'API pensata per essere riusata altrove"
- **Evitare la "ripetizione" nei nomi**: dentro il pacchetto `user`, chiamare un tipo
  `UserRepository` sarebbe ridondante — da fuori si leggerebbe `user.UserRepository`, col
  nome "user" ripetuto due volte. La convenzione Go è lasciare che sia il **nome del
  pacchetto** a dare il contesto: il tipo si chiama solo `Repository`, e da fuori si legge
  `user.Repository` — chiaro e senza ripetizioni

---

### Passo 1 — Aggiungi le dipendenze
Da `backend/`:
```powershell
cd backend
go get golang.org/x/crypto/bcrypt
go get github.com/google/uuid
```
`bcrypt` per l'hashing della password, `uuid` per generare l'ID univoco di ogni utente (lo
stesso tipo di ID, `UUID`, che useremo come chiave primaria quando arriveremo a Postgres).

### Passo 2 — Il modello dati: `internal/user/user.go`
Crea (o correggi, se hai già il file con `package auth`) `backend/services/auth-service/internal/user/user.go`:
```go
package user

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
```
Il nome del pacchetto (`package user`) deve combaciare con il nome della cartella
(`internal/user/`) — non è obbligatorio per il compilatore, ma è la convenzione che tutti i
progetti Go seguono, quindi seguila anche tu. Questo file contiene **solo** la forma dei dati,
niente comportamento.

### Passo 3 — Il repository: `internal/user/user_repository.go`
Crea `backend/services/auth-service/internal/user/user_repository.go`:
```go
package user

import (
	"errors"
	"sync"
)

var ErrAlreadyExists = errors.New("utente già esistente")

type Repository interface {
	Create(u User) error
	GetByEmail(email string) (User, bool)
}

type InMemoryRepository struct {
	mu    sync.Mutex
	users map[string]User // chiave: email
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{users: make(map[string]User)}
}

func (r *InMemoryRepository) Create(u User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[u.Email]; exists {
		return ErrAlreadyExists
	}
	r.users[u.Email] = u
	return nil
}

func (r *InMemoryRepository) GetByEmail(email string) (User, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, ok := r.users[email]
	return u, ok
}
```
`InMemoryRepository` implementa l'interfaccia `Repository` **implicitamente** — in Go non
c'è bisogno di scrivere `implements Repository` da nessuna parte: se i metodi combaciano
nella firma, l'interfaccia è soddisfatta automaticamente. Alla Parte 7 aggiungeremo un
`PostgresRepository` in un terzo file di questo stesso pacchetto, senza toccare `user.go` né
questo file.

### Passo 4 — L'handler HTTP: `internal/auth/register.go`
Crea `backend/services/auth-service/internal/auth/register.go`:
```go
package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/LuigiVanacore/go-zakato-shop/services/auth-service/internal/user"
)

type RegisterHandler struct {
	Users user.Repository
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (h *RegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo della richiesta non valido", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email e password sono obbligatorie", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, "la password deve avere almeno 8 caratteri", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("errore hashing password: %v", err)
		http.Error(w, "errore interno", http.StatusInternalServerError)
		return
	}

	newUser := user.User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}

	if err := h.Users.Create(newUser); err != nil {
		if errors.Is(err, user.ErrAlreadyExists) {
			http.Error(w, "utente già registrato", http.StatusConflict)
			return
		}
		log.Printf("errore salvataggio utente: %v", err)
		http.Error(w, "errore interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(registerResponse{ID: newUser.ID, Email: newUser.Email})
}
```
Punti chiave:
- Importiamo `internal/user` con il suo path completo — da qui in poi, nel codice, `user.`
  è il pacchetto e basta (es. `user.User`, `user.Repository`)
- La variabile locale si chiama `newUser`, non `user`: se l'avessimo chiamata `user` avrebbe
  **nascosto** il nome del pacchetto importato `user` all'interno della funzione, rendendo
  impossibile scrivere `user.ErrAlreadyExists` più sotto. È un errore comune quando un
  pacchetto e una variabile hanno lo stesso nome naturale — tienilo a mente
- Validiamo **prima** di fare qualunque cosa costosa (l'hashing bcrypt è deliberatamente
  lento — meglio scartare subito input palesemente invalido)
- La risposta (`registerResponse`) **non contiene mai** `PasswordHash` — costruiamo
  esplicitamente una struct diversa, minimale, per la risposta: un modo semplice per evitare
  di esporre per errore dati sensibili
- `errors.Is(err, user.ErrAlreadyExists)`: confrontiamo l'errore con una variabile sentinella
  invece che con il testo del messaggio — pratica standard in Go, più robusta

### Passo 5 — Collega tutto in `main.go`
Aggiorna `backend/services/auth-service/cmd/server/main.go`:
```go
package main

import (
	"log"
	"net/http"

	"github.com/LuigiVanacore/go-zakato-shop/services/auth-service/internal/auth"
	"github.com/LuigiVanacore/go-zakato-shop/services/auth-service/internal/user"
)

func main() {
	userRepo := user.NewInMemoryRepository()
	registerHandler := &auth.RegisterHandler{Users: userRepo}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /register", registerHandler.Register)

	addr := ":8080"
	log.Printf("Starting server on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
```
`main()` ora fa da "collante" tra i due pacchetti: crea il repository (`internal/user`), crea
l'handler passandogli il repository (`internal/auth`), registra la route. Nota come `main.go`
non sa **nulla** dei dettagli di `InMemoryRepository` — vede solo l'interfaccia `user.Repository`
attraverso il campo `Users` dell'handler.

### Passo 6 — Aggiorna il `Dockerfile`
Fino ad ora il modulo non aveva dipendenze esterne, quindi `go.sum` non esisteva. Da questo
passo esiste (grazie a `bcrypt` e `uuid`), quindi in
`backend/services/auth-service/Dockerfile` cambia:
```dockerfile
COPY go.mod ./
```
in:
```dockerfile
COPY go.mod go.sum ./
```

### Passo 7 — Prova a mano
Avvia il server (`go run ./services/auth-service/cmd/server` da `backend/`, come nella Parte 1)
e in un altro terminale:
```powershell
curl.exe -X POST http://localhost:8080/register `
  -H "Content-Type: application/json" `
  -d '{"email":"mario@esempio.it","password":"password123"}'
```
(le virgolette **singole** attorno al JSON sono importanti in PowerShell: dentro una stringa
a virgolette singole il testo è letterale, così le virgolette doppie del JSON arrivano intatte
a `curl.exe`. In alternativa, più "nativo" per PowerShell:
`Invoke-RestMethod -Method Post -Uri http://localhost:8080/register -ContentType "application/json" -Body '{"email":"mario@esempio.it","password":"password123"}'`)

Ti aspetti status `201` e un corpo tipo `{"id":"...","email":"mario@esempio.it"}`.

Ora prova a rimandare **la stessa** richiesta: ti aspetti `409` ("utente già registrato").
Prova anche con `password` corta (es. `"123"`): ti aspetti `400`.

### Passo 8 — Test automatico
Crea `backend/services/auth-service/internal/auth/register_test.go`:
```go
package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LuigiVanacore/go-zakato-shop/services/auth-service/internal/user"
)

func TestRegisterHandler(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "registrazione valida",
			body:       `{"email":"mario@esempio.it","password":"password123"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "email mancante",
			body:       `{"password":"password123"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "password troppo corta",
			body:       `{"email":"luigi@esempio.it","password":"corta"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &RegisterHandler{Users: user.NewInMemoryRepository()}

			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			handler.Register(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status atteso %d, ricevuto %d (body: %s)", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRegisterHandler_EmailDuplicata(t *testing.T) {
	repo := user.NewInMemoryRepository()
	handler := &RegisterHandler{Users: repo}
	body := `{"email":"mario@esempio.it","password":"password123"}`

	req1 := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	rec1 := httptest.NewRecorder()
	handler.Register(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("prima registrazione fallita: status %d", rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	rec2 := httptest.NewRecorder()
	handler.Register(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Errorf("status atteso %d per email duplicata, ricevuto %d", http.StatusConflict, rec2.Code)
	}
}
```
Novità rispetto alla Parte 2: un **test tabellare** (*table-driven test*) — invece di scrivere
una funzione `Test...` per ogni caso, elenchiamo i casi in uno slice di struct e li eseguiamo
tutti nello stesso ciclo, con `t.Run(tt.name, ...)` a creare un sotto-test per ciascuno (utile
per vedere subito, nell'output, **quale** caso specifico è fallito). Nota anche che ogni
sotto-test crea un **nuovo** `user.NewInMemoryRepository()`: così i test non si influenzano a
vicenda condividendo stato.

Esegui (da `backend/`):
```powershell
go test ./... -v
```

### Problemi comuni
- **`undefined: user.NewInMemoryRepository`**: controlla il nome del pacchetto in
  `internal/user/*.go` — deve essere `package user` (non più `package auth`, se avevi
  cominciato prima di questa riorganizzazione)
- **`import cycle not allowed`**: se per errore fai importare `internal/auth` da dentro
  `internal/user`, ottieni un ciclo (`user` → `auth` → `user`). Il verso giusto è uno solo:
  `internal/auth` importa `internal/user`, mai il contrario
- **`curl.exe` restituisce un errore di parsing JSON**: quasi sempre un problema di
  quoting in PowerShell — ricontrolla che stai usando virgolette **singole** attorno al JSON
- **Il test di email duplicata fallisce**: assicurati di usare **due richieste separate**
  (`req1`/`req2`) verso **lo stesso** `handler`/`repo` — se ricrei il repository tra le due
  chiamate, la seconda non troverà mai un duplicato
- **`go.sum` non si aggiorna**: esegui `go get` (Passo 1) da dentro `backend/`, non dalla root
  del repository

### Checklist di fine parte
- [ ] `internal/user/user.go` e `internal/user/user_repository.go` sono entrambi `package user`
- [ ] `POST /register` con dati validi risponde `201` e un JSON con `id` ed `email` (mai la password)
- [ ] La stessa registrazione ripetuta risponde `409`
- [ ] Password troppo corta o email mancante rispondono `400`
- [ ] `go test ./... -v` (da `backend/`) mostra tutti i sotto-test passati
- [ ] Hai capito perché `internal/auth` importa `internal/user` e non il contrario

Quando questa checklist è verde, siamo pronti per la **Parte 5** (login: verifica password,
generazione del JWT) — la scrivo in questa cartella quando mi dici che sei arrivato fin qui.

> [← Indice della guida](README.md) · [← Parte 3](03-containerizzazione.md)
