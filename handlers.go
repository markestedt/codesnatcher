package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *Image `json:"image_url,omitempty"`
}

type Image struct {
	Url    string `json:"url,omitempty"`
	Base64 string `json:"base64,omitempty"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type ExtractedCode struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Tag      string `json:"tag"`
}

type CodeResponse struct {
	Code string `json:"code"`
}

func (app *application) getIndexHandler(w http.ResponseWriter, r *http.Request) {
	app.templates.ExecuteTemplate(w, "index", nil)
}

func (app *application) postImageHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	// Write straight to disk
	err := r.ParseMultipartForm(0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	// We always want to remove the multipart file as we're copying
	// the contents to another file anyway
	defer func() {
		if remErr := r.MultipartForm.RemoveAll(); remErr != nil {
			// Log error?
		}
	}()

	// Start reading multi-part file under id "fileupload"
	f, _, err := r.FormFile("image")
	if err != nil {
		if err == http.ErrMissingFile {
			http.Error(w, "Request did not contain a file", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		return
	}
	defer f.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, f); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	code, err := extractCodeFromImage(buf.Bytes())

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	json.NewEncoder(w).Encode(code)
}

func extractCodeFromImage(imageData []byte) (*ExtractedCode, error) {
	// Encode image to base64
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	// Construct the API request
	request := OpenAIRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{
						Type: "text",
						Text: `Extract the code from the provided image and determine the programming language. Consider syntax, keywords, and formatting to ensure accuracy.
								Map the language to an appropriate tag from the list below. The list has the language first and the tags second, separated by hyphen.
								If there is an exact match for the language name you have identified in the list, use that.
                               Return ONLY a JSON object in this exact format (no markdown, no code blocks, no explanations):
                               {
                                   "language": "the programming language name",
                                   "code": "the exact code from the image",
								   "tag": "the tag that matches the programming language name, selected from the list below"
                               }
								   Escape any special characters in the code properly for JSON.
								   ### Language list start ###
								   ` + languages,
					},
					{
						Type: "image_url",
						ImageURL: &Image{
							Url: fmt.Sprintf("data:image/jpeg;base64,%s", base64Data),
						},
					},
				},
			},
		},
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("OPENAI_API_KEY")))

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResponse OpenAIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	// Check for API error
	if apiResponse.Error != nil {
		return nil, fmt.Errorf("API error: %s", apiResponse.Error.Message)
	}

	// Check if we have any choices
	if len(apiResponse.Choices) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	var extractedCode ExtractedCode
	if err := json.Unmarshal([]byte(apiResponse.Choices[0].Message.Content), &extractedCode); err != nil {
		return nil, fmt.Errorf("error parsing structured response: %v", err)
	}

	return &extractedCode, nil
}

var languages string = `
Markup - markup, html, xml, svg, mathml, ssml, atom, rss
CSS - css
C-like - clike
JavaScript - javascript, js
ABAP - abap
ABNF - abnf
ActionScript - actionscript
Ada - ada
Agda - agda
AL - al
ANTLR4 - antlr4, g4
Apache Configuration - apacheconf
Apex - apex
APL - apl
AppleScript - applescript
AQL - aql
Arduino - arduino, ino
ARFF - arff
ARM Assembly - armasm, arm-asm
Arturo - arturo, art
AsciiDoc - asciidoc, adoc
ASP.NET (C#) - aspnet
6502 Assembly - asm6502
Atmel AVR Assembly - asmatmel
AutoHotkey - autohotkey
AutoIt - autoit
AviSynth - avisynth, avs
Avro IDL - avro-idl, avdl
AWK - awk, gawk
Bash - bash, sh, shell
BASIC - basic
Batch - batch
BBcode - bbcode, shortcode
BBj - bbj
Bicep - bicep
Birb - birb
Bison - bison
BNF - bnf, rbnf
BQN - bqn
Brainfuck - brainfuck
BrightScript - brightscript
Bro - bro
BSL (1C:Enterprise) - bsl, oscript
C - c
C# - csharp, cs, dotnet
C++ - cpp
CFScript - cfscript, cfc
ChaiScript - chaiscript
CIL - cil
Cilk/C - cilkc, cilk-c
Cilk/C++ - cilkcpp, cilk-cpp, cilk
Clojure - clojure
CMake - cmake
COBOL - cobol
CoffeeScript - coffeescript, coffee
Concurnas - concurnas, conc
Content-Security-Policy - csp
Cooklang - cooklang
Coq - coq
Crystal - crystal
CSS Extras - css-extras
CSV - csv
CUE - cue
Cypher - cypher
D - d
Dart - dart
DataWeave - dataweave
DAX - dax
Dhall - dhall
Diff - diff
Django/Jinja2 - django, jinja2
DNS zone file - dns-zone-file, dns-zone
Docker - docker, dockerfile
DOT (Graphviz) - dot, gv
EBNF - ebnf
EditorConfig - editorconfig
Eiffel - eiffel
EJS - ejs, eta
Elixir - elixir
Elm - elm
Embedded Lua templating - etlua
ERB - erb
Erlang - erlang
Excel Formula - excel-formula, xlsx, xls
F# - fsharp
Factor - factor
False - false
Firestore security rules - firestore-security-rules
Flow - flow
Fortran - fortran
FreeMarker Template Language - ftl
GameMaker Language - gml, gamemakerlanguage
GAP (CAS) - gap
G-code - gcode
GDScript - gdscript
GEDCOM - gedcom
gettext - gettext, po
Gherkin - gherkin
Git - git
GLSL - glsl
GN - gn, gni
GNU Linker Script - linker-script, ld
Go - go
Go module - go-module, go-mod
Gradle - gradle
GraphQL - graphql
Groovy - groovy
Haml - haml
Handlebars - handlebars, hbs, mustache
Haskell - haskell, hs
Haxe - haxe
HCL - hcl
HLSL - hlsl
Hoon - hoon
HTTP - http
HTTP Public-Key-Pins - hpkp
HTTP Strict-Transport-Security - hsts
IchigoJam - ichigojam
Icon - icon
ICU Message Format - icu-message-format
Idris - idris, idr
.ignore - ignore, gitignore, hgignore, npmignore
Inform 7 - inform7
Ini - ini
Io - io
J - j
Java - java
JavaDoc - javadoc
JavaDoc-like - javadoclike
Java stack trace - javastacktrace
Jexl - jexl
Jolie - jolie
JQ - jq
JSDoc - jsdoc
JS Extras - js-extras
JSON - json, webmanifest
JSON5 - json5
JSONP - jsonp
JS stack trace - jsstacktrace
JS Templates - js-templates
Julia - julia
Keepalived Configure - keepalived
Keyman - keyman
Kotlin - kotlin, kt, kts
KuMir (КуМир) - kumir, kum
Kusto - kusto
LaTeX - latex, tex, context
Latte - latte
Less - less
LilyPond - lilypond, ly
Liquid - liquid
Lisp - lisp, emacs, elisp, emacs-lisp
LiveScript - livescript
LLVM IR - llvm
Log file - log
LOLCODE - lolcode
Lua - lua
Magma (CAS) - magma
Makefile - makefile
Markdown - markdown, md
Markup templating - markup-templating
Mata - mata
MATLAB - matlab
MAXScript - maxscript
MEL - mel
Mermaid - mermaid
METAFONT - metafont
Mizar - mizar
MongoDB - mongodb
Monkey - monkey
MoonScript - moonscript, moon
N1QL - n1ql
N4JS - n4js, n4jsd
Nand To Tetris HDL - nand2tetris-hdl
Naninovel Script - naniscript, nani
NASM - nasm
NEON - neon
Nevod - nevod
nginx - nginx
Nim - nim
Nix - nix
NSIS - nsis
Objective-C - objectivec, objc
OCaml - ocaml
Odin - odin
OpenCL - opencl
OpenQasm - openqasm, qasm
Oz - oz
PARI/GP - parigp
Parser - parser
Pascal - pascal, objectpascal
Pascaligo - pascaligo
PATROL Scripting Language - psl
PC-Axis - pcaxis, px
PeopleCode - peoplecode, pcode
Perl - perl
PHP - php
PHPDoc - phpdoc
PHP Extras - php-extras
PlantUML - plant-uml, plantuml
PL/SQL - plsql
PowerQuery - powerquery, pq, mscript
PowerShell - powershell
Processing - processing
Prolog - prolog
PromQL - promql
.properties - properties
Protocol Buffers - protobuf
Pug - pug
Puppet - puppet
Pure - pure
PureBasic - purebasic, pbfasm
PureScript - purescript, purs
Python - python, py
Q# - qsharp, qs
Q (kdb+ database) - q
QML - qml
Qore - qore
R - r
Racket - racket, rkt
Razor C# - cshtml, razor
React JSX - jsx
React TSX - tsx
Reason - reason
Regex - regex
Rego - rego
Ren'py - renpy, rpy
ReScript - rescript, res
reST (reStructuredText) - rest
Rip - rip
Roboconf - roboconf
Robot Framework - robotframework, robot
Ruby - ruby, rb
Rust - rust
SAS - sas
Sass (Sass) - sass
Sass (SCSS) - scss
Scala - scala
Scheme - scheme
Shell session - shell-session, sh-session, shellsession
Smali - smali
Smalltalk - smalltalk
Smarty - smarty
SML - sml, smlnj
Solidity (Ethereum) - solidity, sol
Solution file - solution-file, sln
Soy (Closure Template) - soy
SPARQL - sparql, rq
Splunk SPL - splunk-spl
SQF: Status Quo Function (Arma 3) - sqf
SQL - sql
Squirrel - squirrel
Stan - stan
Stata Ado - stata
Structured Text (IEC 61131-3) - iecst
Stylus - stylus
SuperCollider - supercollider, sclang
Swift - swift
Systemd configuration file - systemd
T4 templating - t4-templating
T4 Text Templates (C#) - t4-cs, t4
T4 Text Templates (VB) - t4-vb
TAP - tap
Tcl - tcl
Template Toolkit 2 - tt2
Textile - textile
TOML - toml
Tremor - tremor, trickle, troy
Turtle - turtle, trig
Twig - twig
TypeScript - typescript, ts
TypoScript - typoscript, tsconfig
UnrealScript - unrealscript, uscript, uc
UO Razor Script - uorazor
URI - uri, url
V - v
Vala - vala
VB.Net - vbnet
Velocity - velocity
Verilog - verilog
VHDL - vhdl
vim - vim
Visual Basic - visual-basic, vb, vba
WarpScript - warpscript
WebAssembly - wasm
Web IDL - web-idl, webidl
WGSL - wgsl
Wiki markup - wiki
Wolfram language - wolfram, mathematica, nb, wl
Wren - wren
Xeora - xeora, xeoracube
XML doc (.net) - xml-doc
Xojo (REALbasic) - xojo
XQuery - xquery
YAML - yaml, yml
YANG - yang
Zig - zig
`
