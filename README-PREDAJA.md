<!-- cspell:words clusterctl preempciju -->

# CS320 projektni zadatak

## Osnovni podaci

- Tema: Upravljanje procesima u klaster okruženju
- Student: Luka Golubović, indeks 6356
- Implementacija: Go 1.26
- Repozitorijum: <https://github.com/golubovicluka/CS320-cluster-process-manager>
- Identitet izvornog koda i rezultata: `docs/results/evidence-manifest.json`
- Commit izvornog izdanja: `SOURCE_COMMIT.txt`

## Šta je implementirano

Projekat simulira kontrolu procesa nad skupom čvorova sa ograničenim CPU i
memorijskim kapacitetom. Podržani su Round Robin, Least Loaded i Priority-Aware
raspoređivači. Simulator obrađuje životni ciklus procesa, zakazane kvarove
čvorova i procesa, ograničene restart pokušaje, heartbeat timeout, ponovno
raspoređivanje i izvoz metrika u JSON ili CSV.

Repozitorijum sadrži tri izvršna programa:

- `cmd/server` pokreće REST API;
- `cmd/clusterctl` upravlja serverom iz terminala;
- `cmd/simulator` izvršava determinističke JSON scenarije.

## Pokretanje provera

Potreban je Go 1.26 ili noviji. Projekat koristi samo Go standardnu biblioteku.

```bash
make fmt-check
make vet
make test
make race
make build
```

Komanda `make evidence` ponavlja ove provere, regeneriše rezultate i pravi
manifest reproduktivnosti. U predajnom paketu čita commit iz fajla
`SOURCE_COMMIT.txt`, jer arhiva namerno ne sadrži `.git` direktorijum.

## Brza demonstracija

Pokretanje scenarija sa kvarom čvora:

```bash
go run ./cmd/simulator -scenario scenarios/node-failure.json -format json
```

Poređenje dva raspoređivača nad istim heterogenim opterećenjem:

```bash
go run ./cmd/simulator -scenario scenarios/heterogeneous.json -scheduler round-robin
go run ./cmd/simulator -scenario scenarios/heterogeneous.json -scheduler least-loaded
```

REST varijanta se pokreće komandom:

```bash
go run ./cmd/server
```

U drugom terminalu:

```bash
go run ./cmd/clusterctl node add --id node-1 --cpu 4 --memory 4096
go run ./cmd/clusterctl process submit \
  --id p1 --cpu 1 --memory 256 --ticks 5 --priority 10
go run ./cmd/clusterctl simulation step --ticks 5
go run ./cmd/clusterctl report show
```

## Sadržaj predajnog paketa

- formalna projektna dokumentacija u DOCX i PDF formatu;
- izvorni kod aplikacije;
- pet ponovljivih scenario fajlova i sedam referentnih pokretanja;
- verzionisani eksperimentalni rezultati iz `docs/results/`;
- tehnička dokumentacija REST API-ja, arhitekture i eksperimenata;
- ovaj README sa komandama za proveru i demonstraciju.

## Granice rešenja

Simulator ne pokreće korisničke programe na udaljenim računarima. CPU i
memorija su numerički resursi modela. Sistem nema distribuirani konsenzus,
trajnu bazu ni autentifikaciju. Izvršavanje je nepreemptivno: Round Robin kruži
kroz čvorove pri smeštanju procesa i ne predstavlja CPU raspoređivanje po kvantu.
