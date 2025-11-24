# Rebuilding the extractor

cd extractor/src     
go build -o ..\dart-extractor.exe .\.

# Initialize Database with correct paths

codeql database init .\db-dart `
--language=dart `
--search-path .\. `
--source-root <your-project-root>

# Index the files from the working dir

codeql database index-files .\db-dart `
--language=dart `
--working-dir <your-project-root> `
--include "**/*.dart" `
--verbosity progress+++ `
--logdir logs

# Finalize the database with indexed files

codeql database finalize --dbscheme=.\ql\lib\dart.dbscheme .\codeql-dart\db-dart