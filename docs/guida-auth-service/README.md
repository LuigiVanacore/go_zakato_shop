# Guida passo-passo: costruire `auth-service`

Questa guida è pensata per chi non ha (ancora) esperienza pratica con Go: ogni parte spiega
non solo *cosa* scrivere ma anche *perché*. Si procede per **iterazioni piccolissime**: ogni
parte produce qualcosa che **funziona davvero e si può verificare**, prima di passare alla
successiva. Non si salta mai avanti.

> Riferimento architettura generale: [../PIANO_GENERALE.md](../PIANO_GENERALE.md).
> Questa guida è il dettaglio pratico di "costruire auth-service", una parte della Fase 1/2
> di [../PIANO_DETTAGLIATO.md](../PIANO_DETTAGLIATO.md).

## Dove viviamo nel repository

Il modulo Go del progetto è già stato inizializzato in `backend/go.mod` (un unico modulo per
tutti i microservizi, come deciso in precedenza). Tutto il codice Go che scriveremo starà
sotto `backend/`, con questa struttura via via che cresce:

```
backend/
  go.mod
  services/
    auth-service/
      cmd/server/main.go   ← punto di ingresso del programma
      internal/            ← codice privato del servizio (arriverà più avanti)
```

**Importante**: tutti i comandi `go ...` di questa guida vanno lanciati da dentro la cartella
`backend/` (è lì che si trova `go.mod`), non dalla root del repository.

## Indice delle parti

Ogni parte è un file separato in questa cartella, numerato in ordine. Solo le parti già
completate hanno un file scritto — le altre sono titoli: le dettaglieremo una alla volta,
quando arriviamo a completare la precedente.

**Nota sull'ordine**: inizialmente Postgres era la Parte 4, subito dopo Docker. Su richiesta
è stato spostato dopo la logica di autenticazione: prima costruiamo registrazione/login/JWT
con uno storage **in-memory** (una mappa Go, dietro un'interfaccia repository), poi
sostituiamo quello storage con Postgres senza toccare il resto — è il *repository pattern*.

| # | Parte | Stato |
|---|-------|-------|
| 1 | [Server Go minimo con `/health`](01-server-minimo.md) | ✅ pronta |
| 2 | [Test automatico per `/health`](02-test-automatico.md) | ✅ pronta |
| 3 | [Containerizzazione con Docker](03-containerizzazione.md) | ✅ pronta |
| 4 | Registrazione (`POST /register`) con storage in-memory: hashing password, repository pattern | da scrivere |
| 5 | Login (`POST /login`) con storage in-memory: verifica password, generazione JWT | da scrivere |
| 6 | Middleware di autenticazione: proteggere un endpoint col JWT, refresh token | da scrivere |
| 7 | Postgres: sostituire lo storage in-memory con il database vero (Docker Compose, tabella `users`) | da scrivere |
| 8 | Conversione da REST a gRPC (quando arriveremo all'API Gateway) | da scrivere |

Quando completi la checklist di fine parte in un file, dimmelo e scrivo la parte successiva.
