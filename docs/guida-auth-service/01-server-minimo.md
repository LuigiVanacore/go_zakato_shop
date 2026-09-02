# Parte 1 — Server Go minimo con `/health`

> [← Indice della guida](README.md)

### Obiettivo
Un programma Go che, avviato, resta in ascolto su una porta e risponde `200 OK` quando lo
si interroga su `/health`. Niente database, niente logica di business: solo la "presa di
corrente" del servizio, quella che poi Kubernetes userà per sapere se il servizio è vivo.

### Concetti che userai in questo passo
- **Modulo Go**: un progetto Go con le sue dipendenze, dichiarato in `go.mod` (già fatto)
- **Package**: un raggruppamento di file Go. `package main` è speciale: dice a Go "questo è
  un programma eseguibile, non una libreria"
- **`func main()`**: il punto in cui l'esecuzione del programma comincia — esattamente come
  in C, Java, ecc.
- **`net/http`**: il pacchetto della libreria standard di Go per fare server/client HTTP.
  Non serve installare nulla, è già incluso in Go

### Passo 1 — Verifica di avere Go installato
```powershell
go version
```
Deve stampare qualcosa tipo `go version go1.26.x windows/amd64`. Se il comando non è
riconosciuto, Go non è installato: fermati e installa Go da https://go.dev/dl/ prima di
proseguire.

### Passo 2 — Crea le cartelle
Dentro `backend/`, crea questo percorso (puoi farlo dal tuo editor, o da terminale):
```
backend/services/auth-service/cmd/server/
```

### Passo 3 — Scrivi `main.go`
Crea il file `backend/services/auth-service/cmd/server/main.go` con questo contenuto:

```go
package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)

	addr := ":8080"
	log.Printf("auth-service in ascolto su %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
```

Cosa fa, riga per riga:
- `import (...)`: dichiara quali pacchetti della libreria standard usiamo (`log` per stampare
  messaggi, `net/http` per il server HTTP)
- `http.NewServeMux()`: crea un "router" — un oggetto che sa smistare le richieste HTTP in
  arrivo verso la funzione giusta in base al metodo e al path
- `mux.HandleFunc("GET /health", healthHandler)`: dice "quando arriva una GET su `/health`,
  chiama la funzione `healthHandler`"
- `http.ListenAndServe(addr, mux)`: avvia davvero il server, in ascolto sulla porta `8080`,
  e blocca l'esecuzione lì finché il programma non viene fermato o va in errore
- `healthHandler(w http.ResponseWriter, r *http.Request)`: la firma che **ogni** handler HTTP
  in Go deve avere — `w` è dove scrivi la risposta, `r` è la richiesta ricevuta in ingresso
- `w.WriteHeader(http.StatusOK)`: imposta lo status code della risposta a `200`
- `w.Write([]byte("ok"))`: scrive il corpo della risposta (in Go le stringhe si convertono
  in `[]byte` per essere scritte su uno stream)

### Passo 4 — Esegui il programma
Da terminale:
```powershell
cd backend
go run ./services/auth-service/cmd/server
```
`go run` compila e avvia il programma in un solo comando (comodo in sviluppo; più avanti
useremo `go build` per creare un eseguibile vero e proprio, ad es. per Docker).

Se tutto va bene, il terminale resta "bloccato" e mostra:
```
auth-service in ascolto su :8080
```
Questo è normale: il server sta girando e aspetta richieste. Non chiudere il terminale.

### Passo 5 — Verifica che funzioni
Apri un **secondo** terminale (lascia il primo con il server acceso) e prova:
```powershell
curl.exe http://localhost:8080/health
```
(su Windows PowerShell, `curl` da solo è un alias di `Invoke-WebRequest` con output diverso:
usa `curl.exe` per il comportamento classico, oppure apri semplicemente
`http://localhost:8080/health` nel browser)

Ti aspetti: risposta `ok` con status `200`.

Per fermare il server: torna al primo terminale e premi `Ctrl+C`.

### Problemi comuni
- **`bind: address already in use`**: la porta 8080 è già occupata da un altro processo.
  Cambia temporaneamente `":8080"` in `":8081"` nel codice, oppure chiudi il processo che
  la occupa
- **Errori di compilazione tipo `package main is not in std`**: probabilmente il file non è
  nella cartella giusta, o non hai lanciato `go run` da dentro `backend/`
- **`go: cannot find main module`**: sei fuori da `backend/` (dove sta `go.mod`) — fai `cd backend`

### Checklist di fine parte
- [ ] `go run ./services/auth-service/cmd/server` parte senza errori
- [ ] `curl.exe http://localhost:8080/health` (o il browser) risponde `ok` con status 200
- [ ] Hai capito cosa fa ciascuna riga di `main.go` (se qualcosa non è chiaro, chiedimelo
      prima di andare avanti)

Quando questa checklist è verde, siamo pronti per la **Parte 2** (test automatico) — la scrivo
in questa cartella quando mi dici che sei arrivato fin qui.

> [← Indice della guida](README.md)
