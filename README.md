# Introduction
Welcome to codeql-dart, a free-to-use Dart/Flutter Extractor for CodeQL.

## Rebuilding the extractor

```
cd extractor/src     
go build -o ..\dart-extractor.exe .\.
```

## 1. Initialize Database with correct paths

```
codeql database init .\db-dart `
--language=dart `
--search-path .\extractor `
--source-root <your-project-root>
```

## 2. Index the files from the working dir

```
codeql database index-files .\db-dart `
--language=dart `
--working-dir <your-project-root> `
--include "**/*.dart" `
--verbosity progress+++ `
--logdir logs
```

## 3. Finalize the database with indexed files

```
codeql database finalize --dbscheme=.\dart\dart.dbscheme .\db-dart
```