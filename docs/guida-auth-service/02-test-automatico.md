# Parte 2 — Test automatico per `/health`

> [← Indice della guida](README.md) · [← Parte 1](01-server-minimo.md)

### Obiettivo
Finora hai verificato che `/health` funzioni avviando il server a mano e interrogandolo con
`curl`/browser. Va bene per una prova al volo, ma non è ripetibile: da qui in poi il codice
crescerà (registrazione, login, JWT...) e non vuoi riverificare tutto a mano ogni volta. Un
**test automatico** fa questo controllo per te in una frazione di secondo, ogni volta che lo
lanci.

### Concetti che userai in questo passo
- **`go test`**: il comando che cerca ed esegue i test nel progetto
- **File `_test.go`**: Go riconosce come "file di test" solo quelli che finiscono in
  `_test.go`. Vivono nella stessa cartella del codice che testano
- **Pacchetto `testing`**: libreria standard con tutto il necessario per scrivere test
- **Pacchetto `net/http/httptest`**: fornisce finte richieste/risposte HTTP, senza dover
  aprire davvero una porta di rete — il test è quindi velocissimo e non rischia conflitti di
  porta con altri processi
- **Testare l'handler direttamente**: invece di avviare tutto il server e fare una vera
  chiamata HTTP a `localhost:8080`, chiamiamo `healthHandler` come una normale funzione Go,
  passandogli una richiesta e una risposta finte. Stiamo testando *la logica*, non la rete

### Passo 1 — Crea il file di test
Crea `backend/services/auth-service/cmd/server/main_test.go` (stessa cartella di `main.go`,
stesso `package main` — così il test può chiamare `healthHandler` direttamente, anche se non
è esportata con la lettera maiuscola):

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code atteso %d, ricevuto %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if body != "OK" {
		t.Errorf("corpo della risposta atteso %q, ricevuto %q", "OK", body)
	}
}
```

Cosa fa, riga per riga:
- `func TestHealthHandler(t *testing.T)`: ogni funzione di test deve iniziare per `Test`,
  avere maiuscola dopo, e prendere `t *testing.T` — è così che `go test` la riconosce ed
  esegue
- `httptest.NewRequest(...)`: costruisce una richiesta HTTP finta (metodo `GET`, path
  `/health`, nessun body) senza passare dalla rete
- `httptest.NewRecorder()`: un "finto" `http.ResponseWriter` che, invece di spedire la
  risposta a un client vero, la registra in memoria così puoi ispezionarla dopo
- `healthHandler(rec, req)`: chiamiamo l'handler esattamente come farebbe il router — solo
  che qui lo facciamo noi a mano, in un test
- `t.Errorf(...)`: se la condizione è falsa, segnala il test come fallito con quel messaggio
  e **continua** l'esecuzione (utile per vedere più problemi in un colpo solo). Esiste anche
  `t.Fatalf`, che invece interrompe subito il test — la useremo quando un errore rende inutile
  continuare

### Passo 2 — Esegui il test
Da terminale, dentro `backend/`:
```powershell
cd backend
go test ./...
```
`./...` dice "cerca test in questa cartella e in tutte le sottocartelle" — comodo perché non
devi ricordare il percorso esatto ogni volta.

Output atteso:
```
ok  	github.com/LuigiVanacore/go-zakato-shop/services/auth-service/cmd/server	0.XXXs
```
`ok` = tutti i test in quel pacchetto sono passati. Per vedere il nome di ogni singolo test
eseguito, aggiungi `-v`:
```powershell
go test ./... -v
```

### Passo 3 (facoltativo, consigliato) — Guarda un test fallire
Vale la pena vedere anche un fallimento, per riconoscerlo quando capiterà per davvero.
Cambia temporaneamente in `main.go` la riga:
```go
w.Write([]byte("OK"))
```
in:
```go
w.Write([]byte("KO"))
```
e rilancia `go test ./...`. Dovresti vedere qualcosa come:
```
--- FAIL: TestHealthHandler (0.00s)
    main_test.go:16: corpo della risposta atteso "OK", ricevuto "KO"
FAIL
```
Nota come il messaggio ti dice **esattamente** file, riga e cosa si aspettava. Ripristina
`"OK"` e rilancia il test per tornare al verde prima di proseguire.

### Problemi comuni
- **`go test` non trova nessun test**: controlla che il file finisca davvero in `_test.go` e
  sia nella stessa cartella di `main.go`
- **`undefined: healthHandler`**: il file di test non è nello stesso `package main` di
  `main.go`, o è nella cartella sbagliata
- **Il test compila ma non parte**: verifica che il nome della funzione inizi per `Test` con
  la T maiuscola

### Checklist di fine parte
- [ ] `main_test.go` creato accanto a `main.go`
- [ ] `go test ./...` (da dentro `backend/`) stampa `ok`
- [ ] Hai visto anche un test fallire almeno una volta, e capito come leggere il messaggio
- [ ] Hai capito perché testiamo l'handler direttamente invece di avviare il server vero

Quando questa checklist è verde, siamo pronti per la **Parte 3** (containerizzazione con
Docker) — la scrivo in questa cartella quando mi dici che sei arrivato fin qui.

> [← Indice della guida](README.md) · [← Parte 1](01-server-minimo.md)
