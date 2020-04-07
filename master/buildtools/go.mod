module buildtools

require (
	github.com/fatih/color v1.7.0 // indirect
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mattn/go-isatty v0.0.6 // indirect
	github.com/rakyll/gotest v0.0.0-20180125184505-86f0749cd8cc
	golang.org/x/tools v0.0.0-20190228180612-4a0f391d88ad
)

replace golang.org/x/tools => github.com/determined-ai/tools v0.0.0-20190710235009-235279ca75c1

go 1.13
