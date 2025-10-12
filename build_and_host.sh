#!/bin/bash

# Exit on error
set -e

# --- Configuration ---
GO_COMPILER="go"
WASM_OUTPUT="main.wasm"
WASM_EXEC_JS="/usr/local/go/lib/wasm/wasm_exec.js"
HTTP_PORT="8080"

# --- Functions ---

# Function to check for required tools
check_requirements() {
  echo "Checking for required tools..."
  if ! command -v "$GO_COMPILER" &> /dev/null; then
    echo "Go compiler not found. Please install Go and make sure it's in your PATH."
    exit 1
  fi
  if ! [ -f "$HOME/ngrok" ]; then
    echo "ngrok not found at $HOME/ngrok. Please make sure it's there."
    exit 1
  fi
  echo "All required tools found."
}

# Function to compile the Go project to WASM
compile_wasm() {
  echo "Compiling the Go project to WASM..."
  if [ -f "$WASM_EXEC_JS" ]; then
    cp "$WASM_EXEC_JS" .
  else
    echo "wasm_exec.js not found at $WASM_EXEC_JS. Please check your Go installation."
    exit 1
  fi
  GOOS=js GOARCH=wasm "$GO_COMPILER" build -o "$WASM_OUTPUT" .
  echo "Compilation successful."
}

# Function to create the index.html file
create_index_html() {
  echo "Creating index.html..."
  cat > index.html <<EOF
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Go Wasm</title>
</head>
<body>

<script src="wasm_exec.js"></script>
<script>
if (!WebAssembly.instantiateStreaming) { // polyfill
    WebAssembly.instantiateStreaming = async (resp, importObject) => {
        const source = await (await resp).arrayBuffer();
        return await WebAssembly.instantiate(source, importObject);
    };
}

const go = new Go();
WebAssembly.instantiateStreaming(fetch("$WASM_OUTPUT"), go.importObject).then((result) => {
    go.run(result.instance);
});
</script>

</body>
</html>
EOF
  echo "index.html created."
}

# Function to start a simple web server
start_server() {
  echo "Starting a simple web server on port $HTTP_PORT..."
  python3 -m http.server "$HTTP_PORT" &
  SERVER_PID=$!
  echo "Web server started with PID: $SERVER_PID"
}

# Function to start ngrok
start_ngrok() {
  echo "Starting ngrok to expose port $HTTP_PORT..."
  "$HOME/ngrok" http "$HTTP_PORT"
}

# --- Main Script ---

check_requirements
compile_wasm
create_index_html
start_server
start_ngrok
