---
name: cs320
description: >-
  Kanonski kontekst, pravila i procedure za CS320 Operativne sisteme na Univerzitetu Metropolitan:
  projekat upravljanja procesima u klasteru (Go simulator), projektna dokumentacija,
  eseji, ispit i pravilo „Glas dokumenta koji čita profesor”. Koristi ovu veštinu
  pri svakom radu u CS320-PZ ili CS320 dokumentaciji.
---

# CS320 - Operativni sistemi (Workspace Skill)

<!-- cspell:words interprocesnu FCFS Rasporedjivanje clusterctl -->

Kanonsko uputstvo za CS320 operativne sisteme na Fakultetu informacionih tehnologija (Univerzitet Metropolitan):
projekat simulatora procesa u klasteru, akademska projektna dokumentacija, eseji i ispit.

## Kada koristiti ovu veštinu

Učitaj ovaj skill (`/cs320`) kada:
- radiš na implementaciji Go simulatora klastera u `/Users/luka/Projects/CS320-PZ/`;
- pišeš, menjaš ili proveravaš projektnu dokumentaciju za CS320 (`draft.md` ili izvedeni DOCX/PDF);
- proveravaš eseje i ispitne materijale za CS320;
- proveravaš usklađenost akademskog teksta sa pravilom „Glas dokumenta koji čita profesor”.

## Pravilo — „Glas dokumenta koji čita profesor”

Ovo pravilo važi za svaki studentski artefakt: `draft.md`, generisani DOCX i PDF, esej, seminarski rad i projektnu dokumentaciju.

- **U telu rada piše se samo o temi i o projektu na kom student radi.** Ništa o tome kako je dokument nastao, kako je pakovan ni šta se sa njim dalje dešava.
- **Zabranjeno u telu rada:**
  - Pominjanje predaje i statusa: „predaja”, „predajni paket”, „predajno stablo”, „predajna arhiva”, „u predajnom paketu”, odobrenje, prijem, ocena, odbrana.
  - Pominjanje mehanike isporuke i dokaznog aparata: ZIP arhive, `SHA256SUMS`, „manifest reproduktivnosti”, zbirni SHA-256 nad skupom fajlova, `SOURCE_COMMIT.txt`, odsustvo `.git` direktorijuma, README fajlovi paketa.
  - Rečenice koje se ograđuju od sopstvenog dokaza („ne obuhvata kompletno stablo”, „nije nezavisna potvrda izvršenja komandi”).
  - Svaki pomen AI alata i generisanja teksta (tekst govori u prvom licu studenta o sopstvenoj implementaciji).
- **Obavezna verifikacija pre nego što se dokument proglasi gotovim:**

```bash
grep -n -i -E "predaj|predajn|manifest reproduktivnosti|SHA-?256|SOURCE_COMMIT|\.git direktorijum|nezavisna potvrda|odbran" <draft.md>
```

Očekivani rezultat je tačno **0 pogodaka**. Isto pokreni i nad tekstom izvučenim iz renderovanog PDF-a (`pdftotext <doc.pdf> - | grep ...`).

## Obavezni redosled provere i izvori istine

1. `/Users/luka/Projects/uni/AGENTS.md` (krovna pravila repozitorijuma)
2. `/Users/luka/Projects/uni/meta/TASKS.md` (kanonski backlog i status zadataka)
3. `/Users/luka/Projects/uni/courses/CS320/SKILL.md` (predmetni kontekst)
4. `/Users/luka/Projects/CS320-PZ/README.md` (tehnička mapa koda)

## Mapa lokacija

| Uloga | Putanja | Opis |
| --- | --- | --- |
| Implementacija | `/Users/luka/Projects/CS320-PZ/` | Go simulator upravljanja procesima u klasteru |
| Tehnička dokumentacija | `/Users/luka/Projects/CS320-PZ/docs/` | REST API, arhitektura i eksperimenti |
| Verzionisani izvor dokumentacije | `/Users/luka/Projects/uni/projects/CS320-cluster-process-manager/` | `metadata.yaml`, `draft.md`, dijagrami, izvori |
| Aktuelni PDF | `/Users/luka/Projects/Dokumentacije/AKTUELNE-VERZIJE/CS320/Projekat/CS320-Luka-Golubovic-6356-PZ.pdf` | Finalni PDF dokument |
| Aktuelni DOCX | `/Users/luka/Projects/Dokumentacije/AKTUELNE-VERZIJE/CS320/Projekat/CS320-Luka-Golubovic-6356-PZ.docx` | Finalni DOCX dokument |
| Eseji | `/Users/luka/Projects/Dokumentacije/AKTUELNE-VERZIJE/CS320/Eseji/` | Esej 01 i Esej 02 (DOCX, PDF, Pages) |
| Predmetni materijali | `/Users/luka/Projects/uni/courses/CS320/` | Lekcije, plan predmeta, zvanično uputstvo |

## Provere i komande

### 1. Go provere simulatora (u `/Users/luka/Projects/CS320-PZ/`)

```bash
make fmt-check
make vet
make test
make race
make build
```

### 2. Deterministički scenariji simulatora

```bash
go run ./cmd/simulator -scenario scenarios/balanced.json
go run ./cmd/simulator -scenario scenarios/heterogeneous.json
go run ./cmd/simulator -scenario scenarios/overload.json
go run ./cmd/simulator -scenario scenarios/node-failure.json
go run ./cmd/simulator -scenario scenarios/priority-workload.json
```

### 3. DOCX Pipeline i renderovanje (u `/Users/luka/Projects/uni/`)

```bash
node tools/metropolitan-doc.js render projects/CS320-cluster-process-manager/metadata.yaml
python3 tools/populate-docx-toc.py <doc.docx> <doc.pdf> --out <next.docx> --font Arial --keep-summary
```
