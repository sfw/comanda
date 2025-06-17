![robot image](comanda-small.jpg)

# COMandA (Chain Of Models and Actions)

Transform and process data with AI-powered workflows. **COMandA** is the command-line LLM orchestration engine that lets you build, chain, and deploy AI processing pipelines using simple YAML configurations.

comanda acts as a robust infrastructure layer for your AI tasks, similar to how Docker provides a platform for containerization. It empowers developers to define, manage, and execute complex LLM-based processes directly from the command line, integrating seamlessly into existing development and deployment environments.

## Why comanda? Key Differentiators

comanda is engineered to be a powerful and developer-centric LLM orchestration tool. Here's how it stands out:

*   **🔧 Command-line Native:** Compiles directly into your system PATH. You can pipe to it or from it.
    *   **Benefit:** Enables seamless integration with your existing scripts, CI/CD pipelines, and development workflows. Manage AI processes with the simplicity of standard command-line utilities.
*   **🤖 Self-improving Workflows:** Create agentic workflows that can generate and optimize new processing pipelines automatically.
    *   **Benefit:** Build sophisticated systems that adapt and learn, allowing for dynamic and intelligent automation of complex tasks.
*   **🌐 Server Mode:** Abstract multiple LLM providers (OpenAI, Anthropic, Google, Ollama, etc.) behind a unified API.
    *   **Benefit:** Achieve consistent and scalable inference across diverse models. Easily switch providers or integrate new models without overhauling your application logic.
*   **📄 Declarative YAML Workflows:** Define complex multi-step AI processes using simple, human-readable YAML.
    *   **Benefit:** Easily version, share, and manage your AI pipelines. Focus on the logic of your workflows, not on boilerplate code.
*   **🚀 Rapid Prototyping & Generation:** Generate executable comanda YAML workflows from natural language prompts.
    *   **Benefit:** Quickly bootstrap new AI tasks and iterate faster, significantly reducing development time for new pipelines.

## 🚀 Quick Start

This guide will walk you through the initial steps to get comanda up and running.

### 1. Installation

First, you need to get the comanda binary. You have a few options:

*   **Download a Pre-built Binary:** The quickest way is to download a binary for your operating system from our [GitHub Releases page](https://github.com/kris-hansen/comanda/releases).
*   **Install via Go:** If you have Go installed, you can use `go install github.com/kris-hansen/comanda@latest`.
*   **Build from Source:** Clone the repository and build it yourself with `go build .` in the repository directory.

For detailed instructions, please refer to the [Installation](#installation) section below.

![Comanda Install Demo](comanda-install.gif)

### 2. Initial Configuration

Once comanda is installed, you need to configure your LLM providers. This tells comanda which models you want to use and provides the necessary API keys. These are stored in an .env file which is by default in the current working directory and can be set with the environment variable see the [Configuration](#configuration) section for more information on this.

Run the following command in your terminal:

```bash
comanda configure
```

This will launch an interactive setup process where you can:

1.  Select an LLM provider (e.g., OpenAI, Anthropic, Google, Ollama).
2.  Enter your API key for that provider (if applicable).
3.  Specify the model name(s) you want to use from that provider.
4.  Choose the mode for each model (text, vision, etc.).

Repeat this for each provider you intend to use. Your configuration, including API keys, will be stored in a `.env` file in your current directory by default. For more advanced configuration options, including encryption, see the [Configuration](#configuration) section.

![Comanda configure demo](comanda-configure.gif)

### 3. Your First comanda Workflow

comanda uses YAML files to define workflows. A workflow consists of one or more steps, where each step performs an action, often involving an LLM.

Let's create a very simple workflow. Create a file named `hello_world.yaml` with the following content:

```yaml
# hello-world.yaml
say_hello:
  input: NA # Input is not applicable because we are generating
  model: gpt-4o  # Or any model you configured, e.g., a local Ollama model
  action: Write a small haiku which includes the words "hello world!"    # this is your prompt
  output: STDOUT     # This will print the output to your terminal
```

**Explanation:**

*   `say_hello`: This is the name of our step, it can be anything as it's really a label for the step name.

*   `input:`: This can be a file in various forms and formats or a database or other input types (see examples). In our case it's NA as we are generating new text.
*   `model`: Specifies which LLM to use. Make sure this matches a model you set up in `comanda configure`.
*   `action`: This is the instruction we give to the LLM.
*   `output:`: This can be a file of various formats or a database. In the case of this simple example it will just be standard output to the terminal.

To run this workflow, use the `process` command:

```bash
comanda process hello_world.yaml
```

You should see the LLM's haiku printed in your terminal!

![Comanda process example](comanda-process.gif)

This is just a basic example. comanda can do much more, including chaining multiple steps, working with files, processing images, and interacting with web content.

Build more robust agentic workflows by:
*  Chaining steps together by having the output of some steps feed the input of other steps
*  Give the steps agentic roles by prompting for context in the action
*  Pass documents as input and output
*  Have some steps assess the work of other steps

Explore the [Features](#features) and [Examples](examples/README.md) to learn more.

## Features

- 🔗 Chain multiple LLM operations together using simple YAML configuration
- 🤖 Support for multiple LLM providers (OpenAI, Anthropic, Google, X.AI, Ollama)
- 📄 File-based operations and transformations
- 🖼️ Support for image analysis with vision models (screenshots and common image formats)
- 🌐 Direct URL input support for web content analysis
- 🕷️ Advanced web scraping capabilities with configurable options
- 🛠️ Support for specialty steps such as OpenAI Responses
- 🚀 Parallel processing of independent steps for improved performance
- 🔒 HTTP server mode: use it as a multi-LLM workflow wrapper
- 🔐 Secure configuration encryption for protecting API keys and secrets
- 📁 Multi-file input support with content consolidation
- 📝 Markdown file support for reusable actions (prompts)
- 🗄️ Database integration for read/write operations for inputs and outputs
- 🔍 Wildcard pattern support for processing multiple files (e.g., `*.pdf`, `data/*.txt`)
- 🛡️ Resilient batch processing with error handling for multiple files
- 📂 Runtime directory support for organizing uploads and YAML processing scripts
- ✨ YAML workflow generation from natural language prompts


## Installation

### Download Pre-built Binary

The easiest way to get started is to download a pre-built binary from the [GitHub Releases page](https://github.com/kris-hansen/comanda/releases). Binaries are available for:
- Windows (x86, amd64)
- macOS (amd64, arm64)
- Linux (x86, amd64, arm64)

Download the appropriate binary for your system, extract it if needed, and place it somewhere in your system's PATH.

### Install via Homebrew (macOS arm)

Mac users can install comanda using Homebrew:

```bash
# Option 1: Tap and install
brew tap kris-hansen/comanda
brew install comanda

# Option 2: Install directly
brew install kris-hansen/comanda/comanda
```

### Install via Go

```bash
go install github.com/kris-hansen/comanda@latest
```

### Build from Source

```bash
git clone https://github.com/kris-hansen/comanda.git
cd comanda
go build
```

## Configuration

### Environment File

comanda uses an environment file to store provider configurations and API keys. By default, it looks for a `.env` file in the current directory. You can specify a custom path using the `COMANDA_ENV` environment variable:

```bash
# Use a specific env file
export COMANDA_ENV=/path/to/your/env/file
comanda process your-workflow-file.yaml

# Or specify it inline
COMANDA_ENV=/path/to/your/env/file comanda process your-workflow-file.yaml
```

### Configuration Encryption

comanda supports encrypting your configuration file to protect sensitive information like API keys. The encryption uses AES-256-GCM with password-derived keys, providing strong security against unauthorized access.

To encrypt your configuration:
```bash
comanda configure --encrypt
```

You'll be prompted to enter and confirm an encryption password. Once encrypted, all commands that need to access the configuration (process, server, configure) will prompt for the password.

Example workflow:
```bash
# First, configure your providers and API keys
comanda configure

# Then encrypt the configuration
comanda configure --encrypt
Enter encryption password: ********
Confirm encryption password: ********
Configuration encrypted successfully!

# When running commands, you'll be prompted for the password
comanda process your-workflow-file.yaml
Enter decryption password: ********
```

The encryption system provides:
- AES-256-GCM encryption (industry standard)
- Password-based key derivation
- Protection against tampering
- Brute-force resistance

You can still view your configuration using:
```bash
comanda configure --list
```
This will prompt for the password if the configuration is encrypted.

### Provider Configuration

Users updating an existing comanda installation may need to run `comanda configure` to select and enable these new models.
A guide for adding new models to existing providers can be found in [docs/adding-new-model-guide.md](docs/adding-new-model-guide.md).

#### OpenAI o1 and o3 Models Support

comanda supports OpenAI's reasoning model families including o1-pro, o1-mini, o3-pro, and o4-mini. These models use the OpenAI Responses API format which is different from the standard Chat Completions API.

**Important Notes:**
- These models may take significantly longer to process requests (up to several minutes for complex prompts)
- The o1-pro and o3-pro models in particular can be much slower than other models
- comanda automatically extends timeouts for these models to prevent connection issues
- When using these models in workflows, expect longer processing times, especially for the first few tokens

Configure your providers and models using the interactive configuration command:

```bash
comanda configure
```

This will prompt you to:

1. Select a provider (OpenAI/Anthropic/Google/X.AI/Ollama)
2. Enter API key (for OpenAI/Anthropic/Google/X.AI)
3. Specify model name
4. Select model mode:
   - text: For text-only operations
   - vision: For image analysis capabilities
   - multi: For both text and image operations

You can view your current configuration using:

```bash
comanda configure --list                       
Server Configuration:
Port: 8080
Data Directory: data
Authentication Enabled: true

Configured Providers:

anthropic:
  - claude-3-5-latest (external)

google:
  - gemini-pro (external)

ollama:
  - llama3.2 (local)

openai:
  - gpt-4o-mini (external)
  - gpt-4o (external)

xai:
  - grok-beta (external)
```

To remove a model from the configuration:

```bash
comanda configure --remove <model-name>
```

To update an API key for a provider (e.g., after key rotation):

```bash
comanda configure --update-key=<provider-name>
```

This will prompt you for the new API key and update it in the configuration. For example:

```bash
comanda configure --update-key=openai
Enter new API key: sk-...
Successfully updated API key for provider 'openai'
```

### Setting the Default Model for Generation

You can set a default model for the `comanda generate` command, which creates YAML workflows from natural language prompts:

```bash
# Interactively set default model
comanda configure --default

# Or set directly
comanda configure --set-default-generation-model gpt-4
```

Once set, you can use `comanda generate` without specifying a model:

```bash
# Uses the default model
comanda generate my_workflow.yaml "Create a workflow to analyze CSV files"

# Override the default with a specific model
comanda generate my_workflow.yaml "Create a workflow to analyze CSV files" --model claude-3-opus
```

When configuring a model that already exists, you'll be prompted to update its mode. This allows you to change a model's capabilities without removing and re-adding it.

Example configuration output:

``` yaml
Configuration from .env:

Server Configuration:
Port: 8088
Data Directory: data
Authentication Enabled: true
Bearer Token: <redacted>

Configured Providers:

ollama:
  - llama2:latest (local)
    Modes: text

openai:
  - gpt-4-turbo-preview (external)
    Modes: text, vision, multi, file
  - gpt-4-vision-preview (external)
    Modes: vision
  - gpt-4o (external)
    Modes: text, vision, multi, file
  - gpt-4o-mini (external)
    Modes: text, vision, multi, file
  - o1-mini (external)
    Modes: text
  - o1-preview (external)
    Modes: text

xai:
  - grok-beta (external)
    Modes: text, file
  - grok-vision-beta (external)
    Modes: vision

anthropic:
  - claude-3-5-sonnet-20241022 (external)
    Modes: text, vision, multi, file
  - claude-3-5-sonnet-latest (external)
    Modes: text, vision, multi, file
  - claude-3-5-haiku-latest (external)
    Modes: text, vision, multi, file
  - claude-opus-4-20250514 (external)
    Modes: text, vision, multi, file
  - claude-sonnet-4-20250514 (external)
    Modes: text, vision, multi, file

Users updating an existing comanda installation may need to run `comanda configure` to select and enable these new models.
A guide for adding new models to existing providers can be found in [docs/adding-new-model-guide.md](docs/adding-new-model-guide.md).

deepseek:
  - deepseek-chat (external)
    Modes: text, vision, multi, file

google:
  - gemini-1.5-flash (external)
    Modes: text, vision, multi, file
  - gemini-1.5-flash-8b (external)
    Modes: text, vision, multi, file
  - gemini-1.5-pro (external)
    Modes: text, vision, multi, file
  - gemini-2.0-flash-exp (external)
    Modes: text, vision, multi, file
  - gemini-2.0-flash-001 (external)
    Modes: text, vision, multi, file
  - gemini-2.0-pro-exp-02-05 (external)
    Modes: text, vision, multi, file
  - gemini-2.0-flash-lite-preview-02-05 (external)
    Modes: text, vision, multi, file
  - gemini-2.0-flash-thinking-exp-01-21 (external)
    Modes: text, vision, multi, file
```

### Server Configuration

comanda can run as an HTTP server, allowing you to process chains of models and actions defined in YAML files via HTTP requests. The server is managed using the `server` command:

```bash
# Start the server
comanda server

# Configure server settings
comanda server configure        # Interactive configuration
comanda server show            # Show current configuration
comanda server port 8080       # Set server port
comanda server datadir ./data  # Set data directory
comanda server auth on         # Enable authentication
comanda server auth off        # Disable authentication
comanda server newtoken        # Generate new bearer token
comanda server cors            # Configure CORS settings
```

The server provides several configuration commands:

- `configure`: Interactive configuration for all server settings including port, data directory, authentication, and CORS
- `show`: Display current server configuration including CORS settings
- `port`: Set the server port
- `datadir`: Set the data directory for YAML files
- `auth`: Enable/disable authentication
- `newtoken`: Generate a new bearer token
- `cors`: Configure CORS settings interactively

The CORS configuration allows you to:
- Enable/disable CORS headers
- Set allowed origins (use * for all, or specify domains)
- Configure allowed HTTP methods
- Set allowed headers
- Define max age for preflight requests

The server configuration is stored in your `.env` file alongside provider and model settings:

```yaml
server:
  port: 8080
  data_dir: "examples"  # Directory containing YAML files to process
  runtime_dir: "runtime"  # Optional directory for runtime files like uploads and YAML processing scripts
  bearer_token: "your-generated-token"
  enabled: true  # Whether authentication is required
  cors:
    enabled: true  # Enable/disable CORS
    allowed_origins: ["*"]  # List of allowed origins, ["*"] for all
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]  # List of allowed HTTP methods
    allowed_headers: ["Authorization", "Content-Type"]  # List of allowed headers
    max_age: 3600  # Max age for preflight requests in seconds
```

The CORS configuration allows you to control Cross-Origin Resource Sharing settings:
- `enabled`: Enable or disable CORS headers (default: true)
- `allowed_origins`: List of origins allowed to access the API. Use `["*"]` to allow all origins, or specify domains like `["https://example.com"]`
- `allowed_methods`: List of HTTP methods allowed for cross-origin requests
- `allowed_headers`: List of headers allowed in requests
- `max_age`: How long browsers should cache preflight request results

### Runtime Directory

The runtime directory feature allows you to organize uploaded files and YAML processing scripts in a dedicated directory within the data directory. This helps keep your server data organized and prevents clutter in the main data directory.

You can specify a runtime directory in the following ways:

1. In the server configuration (as shown above with the `runtime_dir` setting)
2. As a query parameter in API requests:
   ```
   /files/upload?runtimeDir=uploads
   /yaml/upload?runtimeDir=scripts
   /yaml/process?runtimeDir=scripts
   /process?filename=example.yaml&runtimeDir=runtime
   ```

When a runtime directory is specified, the server will:
1. Create the directory if it doesn't exist
2. Store uploaded files in that directory
3. Look for files in that directory first before falling back to the data directory
4. Use that directory for temporary files during processing

This feature is particularly useful for:
- Organizing uploads by type (documents, images, etc.)
- Keeping YAML scripts separate from other files
- Creating separate workspaces for different projects or users
- Isolating temporary files from permanent storage

To start the server:

```bash
comanda server
```

The server provides the following endpoints:

### 1. File Operations

#### View File Contents
```bash
# Get file content as plain text
curl -H "Authorization: Bearer your-token" \
     -H "Accept: text/plain" \
     "http://localhost:8080/files/content?path=example.txt"

# Download binary file
curl -H "Authorization: Bearer your-token" \
     -H "Accept: application/octet-stream" \
     "http://localhost:8080/files/download?path=example.pdf" \
     --output downloaded_file.pdf

# Upload a file
curl -X POST \
     -H "Authorization: Bearer your-token" \
     -F "file=@/path/to/local/file.txt" \
     -F "path=destination/file.txt" \
     "http://localhost:8080/files/upload?runtimeDir=uploads"
```

Using JavaScript:
```javascript
// Get file content
async function getFileContent(path) {
  const response = await fetch(`http://localhost:8080/files/content?path=${encodeURIComponent(path)}`, {
    headers: {
      'Authorization': 'Bearer your-token',
      'Accept': 'text/plain'
    }
  });
  return await response.text();
}

// Download file
async function downloadFile(path) {
  const response = await fetch(`http://localhost:8080/files/download?path=${encodeURIComponent(path)}`, {
    headers: {
      'Authorization': 'Bearer your-token',
      'Accept': 'application/octet-stream'
    }
  });
  const blob = await response.blob();
  // Create download link
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = path.split('/').pop(); // Use filename from path
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(url);
  document.body.removeChild(a);
}

// Upload file
async function uploadFile(file, path) {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('path', path);

  const response = await fetch('http://localhost:8080/files/upload', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer your-token'
    },
    body: formData
  });
  return await response.json();
}
```

### 2. Process Endpoint

`GET /process` processes a YAML file from the configured data directory. For YAML files that use STDIN as their first input, `POST /process` is also supported. Both endpoints support real-time output streaming using Server-Sent Events.

#### GET Request
```bash
# Regular processing (JSON response)
curl "http://localhost:8080/process?filename=openai-example.yaml"

# Streaming processing (Server-Sent Events)
curl -H "Accept: text/event-stream" \
     "http://localhost:8080/process?filename=openai-example.yaml&streaming=true"

# With authentication (when enabled) and runtime directory
curl -H "Authorization: Bearer your-token" \
     -H "Accept: text/event-stream" \
     "http://localhost:8080/process?filename=openai-example.yaml&streaming=true&runtimeDir=runtime"
```

#### POST Request (for YAML files with STDIN input)
You can provide input either through a query parameter or JSON body:

```bash
# Regular processing with query parameter
curl -X POST "http://localhost:8080/process?filename=stdin-example.yaml&input=your text here"

# Regular processing with JSON body
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"input":"your text here", "streaming": false}' \
     "http://localhost:8080/process?filename=stdin-example.yaml"

# Streaming processing with JSON body
curl -X POST \
     -H "Content-Type: application/json" \
     -H "Accept: text/event-stream" \
     -d '{"input":"your text here", "streaming": true}' \
     "http://localhost:8080/process?filename=stdin-example.yaml"
```

Note: POST requests are only allowed for YAML files where the first step uses "STDIN" as input. The /list endpoint shows which methods (GET or GET,POST) are supported for each YAML file.

Response format (non-streaming):
```json
{
  "success": true,
  "message": "Successfully processed openai-example.yaml",
  "output": "Response from gpt-4o-mini:\n..."
}
```

Response format (streaming):
```
data: Processing step 1...

data: Model response: ...

data: Processing step 2...

data: Processing complete
```

Error response (non-streaming):
```json
{
  "success": false,
  "error": "Error message here",
  "output": "Any output generated before the error"
}
```

Using JavaScript:
```javascript
// Regular processing
async function processFile(filename, input = null) {
  const url = `http://localhost:8080/process?filename=${encodeURIComponent(filename)}`;
  const options = {
    method: input ? 'POST' : 'GET',
    headers: {
      'Authorization': 'Bearer your-token',
      'Content-Type': 'application/json'
    }
  };
  
  if (input) {
    options.body = JSON.stringify({ input, streaming: false });
  }
  
  const response = await fetch(url, options);
  return await response.json();
}

// Streaming processing
async function processFileStreaming(filename, input = null) {
  const url = `http://localhost:8080/process?filename=${encodeURIComponent(filename)}`;
  const options = {
    method: input ? 'POST' : 'GET',
    headers: {
      'Authorization': 'Bearer your-token',
      'Content-Type': 'application/json',
      'Accept': 'text/event-stream'
    }
  };
  
  if (input) {
    options.body = JSON.stringify({ input, streaming: true });
  }
  
  const response = await fetch(url, options);
  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    
    const text = decoder.decode(value);
    // Handle each SSE message
    console.log(text);
  }
}
```

### 2. List Endpoint

`GET /list` returns a list of YAML files in the configured data directory, along with their supported HTTP methods:

```bash
curl -H "Authorization: Bearer your-token" "http://localhost:8080/list"
```

Response format:
```json
{
  "success": true,
  "files": [
    {
      "name": "openai-example.yaml",
      "methods": "GET"
    },
    {
      "name": "stdin-example.yaml",
      "methods": "GET,POST"
    }
  ]
}
```
The `methods` field indicates which HTTP methods are supported:
- `GET`: The YAML file can be processed normally
- `GET,POST`: The YAML file accepts STDIN string input via POST request

### 3. Health Check Endpoint

`GET /health` returns the server's current status:

```bash
curl -H "Authorization: Bearer your-token" "http://localhost:8080/health"
```

Response format:
```json
{
  "success": true,
  "message": "Server is healthy",
  "statusCode": 200,
  "response": "OK"
}
```

### 4. YAML Operations

#### Upload YAML
`POST /yaml/upload` uploads a YAML file for processing:

```bash
curl -X POST \
     -H "Authorization: Bearer your-token" \
     -H "Content-Type: application/json" \
     -d '{"content": "your yaml content here"}' \
     "http://localhost:8080/yaml/upload?runtimeDir=myproject"
```

Response format:
```json
{
  "success": true,
  "message": "YAML file uploaded successfully"
}
```

#### Process YAML
`POST /yaml/process` processes a YAML file with optional real-time output streaming:

```bash
# Regular processing (JSON response)
curl -X POST \
     -H "Authorization: Bearer your-token" \
     -H "Content-Type: application/json" \
     -d '{"content": "your yaml content here", "streaming": false}' \
     "http://localhost:8080/yaml/process?runtimeDir=myproject"

# Streaming processing (Server-Sent Events)
curl -X POST \
     -H "Authorization: Bearer your-token" \
     -H "Content-Type: application/json" \
     -H "Accept: text/event-stream" \
     -d '{"content": "your yaml content here", "streaming": true}' \
     "http://localhost:8080/yaml/process"
```

Response format (non-streaming):
```json
{
  "success": true,
  "yaml": "processed yaml content"
}
```

Response format (streaming):
```
data: Processing step 1...

data: Model response: ...

data: Processing step 2...

data: Processing complete
```

Using JavaScript:
```javascript
// Upload YAML
async function uploadYaml(content) {
  const response = await fetch('http://localhost:8080/yaml/upload', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer your-token',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ content })
  });
  return await response.json();
}

// Process YAML (non-streaming)
async function processYaml(content) {
  const response = await fetch('http://localhost:8080/yaml/process', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer your-token',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ content, streaming: false })
  });
  return await response.json();
}

// Process YAML (streaming)
async function processYamlStreaming(content) {
  const response = await fetch('http://localhost:8080/yaml/process', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer your-token',
      'Content-Type': 'application/json',
      'Accept': 'text/event-stream'
    },
    body: JSON.stringify({ content, streaming: true })
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    
    const text = decoder.decode(value);
    // Handle each SSE message
    console.log(text);
  }
}
```

The server logs all requests to the console, including:
- Timestamp
- Request method and path
- Query parameters
- Authorization header (token masked)
- Response status code
- Request duration

Example server log:
```
2024/11/02 21:06:33 Request: method=GET path=/health query= auth=Bearer ******** status=200 duration=875µs
2024/11/02 21:06:37 Request: method=GET path=/list query= auth=Bearer ******** status=200 duration=812.208µs
2024/11/02 21:06:45 Request: method=GET path=/process query=filename=examples/openai-example.yaml auth=Bearer ******** status=200 duration=3.360269792s
```

## Usage

### Supported File Types

comanda supports various file types for input:

- Text files: `.txt`, `.md`, `.yml`, `.yaml`
- Image files: `.png`, `.jpg`, `.jpeg`, `.gif`, `.bmp`
- Web content: Direct URLs to web pages, JSON APIs, or other web resources
- Special inputs: `screenshot` (captures current screen)
- Wildcard patterns: `*.txt`, `data/*.pdf`, etc. to process multiple files at once

When using vision-capable models (like gpt-4o), you can analyze both images and screenshots alongside text content.

Images are automatically optimized for processing:

- Large images are automatically resized to a maximum dimension of 1024px while preserving aspect ratio
- PNG compression is applied to reduce token usage while maintaining quality
- These optimizations help prevent rate limit errors and ensure efficient processing

The screenshot feature allows you to capture the current screen state for analysis. When you specify `screenshot` as the input in your Workflow file, comanda will automatically capture the entire screen and pass it to the specified model for analysis. This is particularly useful for UI analysis, bug reports, or any scenario where you need to analyze the current screen state.

For URL inputs, comanda automatically:

- Detects and validates URLs in input fields
- Fetches content with appropriate error handling
- Handles different content types (HTML, JSON, plain text)
- Stores content in temporary files with appropriate extensions
- Cleans up temporary files after processing

### Creating YAML Workflow Files

Create a YAML file defining your chain of operations:

```yaml
# example.yaml
summarize:
    model: "gpt-4"
    provider: "openai"
    input: 
      file: "input.txt"
    prompt: "Summarize the following content:"
    output:
      file: "summary.txt"

analyze:
    model: "claude-2"
    provider: "anthropic"
    input:
      file: "summary.txt"
    prompt: "Analyze the key points in this summary:"
    output:
      file: "analysis.txt"
```

#### Using Wildcard Patterns

You can use wildcard patterns to process multiple files at once:

```yaml
# wildcard-example.yaml
process-text-files:
  input: 
    - "examples/*.txt"  # Process all text files in the examples directory
  model: "gpt-4o"
  action: "Summarize the content of each file."
  output: "STDOUT"

process-pdf-files:
  input:
    - "examples/document-processing/*.pdf"  # Process all PDF files in the document-processing directory
  model: "gpt-4o"
  action: "Extract key information from each PDF."
  output: "STDOUT"

process-mixed-files:
  input:
    - "examples/file-processing/*.txt"  # Process multiple file types at once
    - "examples/model-examples/*.yaml"
  model: "gpt-4o"
  action: "Analyze the structure and content of each file."
  output: "STDOUT"
```

Wildcard patterns support standard glob syntax:
- `*` matches any number of characters within a filename
- `?` matches a single character
- `[abc]` matches any character in the brackets
- `[a-z]` matches any character in the range

This feature is particularly useful for batch processing multiple files with similar content or for comparing files of the same type.

#### Batch Processing Options

When processing multiple files, you can control how they're handled using batch processing options:

```yaml
# batch-processing-example.yaml
process-files-individually:
  input: 
    - "examples/*.txt"  # Process all text files in the examples directory
  model: "gpt-4o"
  action: "Summarize the content of each file."
  output: "STDOUT"
  batch_mode: "individual"  # Process each file individually (safer than "combined")
  skip_errors: true  # Continue processing even if some files fail
```

The batch processing options include:

- `batch_mode`: Controls how multiple files are processed
  - `individual`: Process each file separately and combine results (safer, default)
  - `combined`: Combine all files into a single prompt (original behavior)
- `skip_errors`: Whether to continue processing if some files fail
  - `true`: Continue processing other files if some fail
  - `false`: Stop processing if any file fails

Individual batch mode is particularly useful when:
- Processing files that might contain encoding issues
- Working with large numbers of files
- Needing to identify which specific files might be problematic

For image analysis:

```yaml
# image-analysis.yaml
analyze:
  input: "image.png"  # Can be any supported image format
  model: "gpt-4o"
  action: "Analyze this image and describe what you see in detail."
  output: "STDOUT"
```

### Parallel Processing

comanda supports parallel processing of independent steps to improve performance. This is particularly useful for tasks that don't depend on each other, such as:

- Running the same prompt against multiple models for comparison
- Processing multiple files independently
- Performing different analyses on the same input

To use parallel processing, define steps under a `parallel-process` block in your YAML file:

```yaml
# parallel-model-comparison.yaml
parallel-process:
  gpt4o_step:
    input:
      - NA
    model: gpt-4o
    action:
      - write a short story about a robot that discovers it has emotions
    output:
      - examples/parallel-processing/gpt4o-story.txt

  claude_step:
    input:
      - NA
    model: claude-3-5-sonnet-latest
    action:
      - write a short story about a robot that discovers it has emotions
    output:
      - examples/parallel-processing/claude-story.txt

compare_step:
  input:
    - examples/parallel-processing/gpt4o-story.txt
    - examples/parallel-processing/claude-story.txt
  model: gpt-4o
  action:
    - compare these two short stories about robots discovering emotions
    - which one is more creative and has better narrative structure?
  output:
    - STDOUT
```

In this example:
- The `gpt4o_step` and `claude_step` will run in parallel
- The `compare_step` will run after both parallel steps complete, as it depends on their outputs

The system automatically validates dependencies between steps to ensure:
- No circular dependencies exist
- Steps that depend on outputs from other steps run after those steps complete
- Parallel steps are truly independent of each other

Parallel processing leverages Go's concurrency features (goroutines and channels) for efficient execution.

### Running Commands

Run your YAML workflow file:

```bash
comanda process your-workflow-file.yaml
```

For example:

```bash
Processing Workflow file: examples/openai-example.yaml

Configuration:

Step: step_one
- Input: [examples/example_filename.txt]
- Model: [gpt-4o-mini]
- Action: [look through these company names and identify the top five which seem most likely in the HVAC business]
- Output: [STDOUT]

Step: step_two
- Input: [STDIN]
- Model: [gpt-4o]
- Action: [for each of these company names provide a snappy tagline that would make them stand out]
- Output: [STDOUT]


Response from gpt-4o-mini:
Based on the company names provided, the following five seem most likely to be in the HVAC (Heating, Ventilation, and Air Conditioning) business:

1. **Evergreen Industries** - The name suggests a focus on sustainability, which is often associated with HVAC systems that promote energy efficiency.

2. **Mountain Peak Investments** - While not directly indicative of HVAC, the name suggests a focus on construction or infrastructure, which often involves HVAC installations.

3. **Cascade Technologies** - The term "cascade" could relate to water systems or cooling technologies, which are relevant in HVAC.

4. **Summit Technologies** - Similar to Mountain Peak, "Summit" may imply involvement in high-quality or advanced systems, possibly including HVAC solutions.

5. **Zenith Industries** - The term "zenith" suggests reaching the highest point, which can be associated with premium or top-tier HVAC products or services.

These names suggest a connection to industries related to heating, cooling, or building systems, which are integral to HVAC.

Response from gpt-4o:
Certainly! Here are some snappy taglines for each of the company names that could help them stand out in the HVAC industry:

1. **Evergreen Industries**: "Sustainability in Every Breath."

2. **Mountain Peak Investments**: "Building Comfort from the Ground Up."

3. **Cascade Technologies**: "Cooling Solutions That Flow."

4. **Summit Technologies**: "Reaching New Heights in HVAC Innovation."

5. **Zenith Industries**: "At the Pinnacle of Climate Control Excellence."
```

## Database Operations

comanda supports database operations as input and output in the YAML workflow. Currently, PostgreSQL is supported.

### Database Configuration

Before using database operations, configure your database connection:

```bash
comanda configure --database
```

This will prompt for:
- Database configuration name (used in YAML files)
- Database type (postgres)
- Host, port, username, password, database name

### Database Input/Output Format

Reading from a database:
```yaml
input:
  database: mydb  # Database configuration name
  sql: SELECT * FROM customers LIMIT 5  # Must be SELECT statement
```

Writing to a database:
```yaml
output:
  database: mydb
  sql: INSERT INTO customers (first_name, last_name, email) VALUES ('John', 'Doe', 'john.doe@example.com')
```

### Example YAML Files
Examples can be found in the `examples/` directory. Here is a link to the README for the examples: [examples/README.md](examples/README.md)

## Project Structure

```bash
comanda/
├── cmd/                    # Command line interface
├── utils/
│   ├── config/            # Configuration handling
│   ├── input/             # Input validation and processing
│   ├── models/            # LLM provider implementations
│   ├── scraper/           # Web scraping functionality
│   └── processor/         # DSL processing logic
├── go.mod
├── go.sum
└── main.go
```

## Roadmap

The following features are being considered:

- More providers:
  - Huggingface inference API?
  - Image generation providers?
  - others?
- URL output support, post this data to URL
  - Need to add credential support
  - Need to solve for local secrets encryption
- Branching and basic if/or logic
- Routing logic i.e., use this model if the output is x and that model if y

## Contributing

Contributions are welcome! Here's how you can help:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please ensure your PR:

- Includes tests for new functionality
- Updates documentation as needed
- Follows the existing code style
- Includes a clear description of the changes

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Citation

If you use comanda in your research or academic work, please cite it as follows:

### BibTeX
```bibtex
@software{comanda2024,
  author       = {Hansen, Kris},
  title        = {comanda: Chain of Models and Actions},
  year         = {2024},
  publisher    = {GitHub},
  url          = {https://github.com/kris-hansen/comanda},
  description  = {A command-line tool for composing Large Language Model operations using YAML-based workflows}
}
```

## Acknowledgments

- OpenAI and Anthropic for their LLM APIs
- The Ollama project for local LLM support
- The Go community for excellent libraries and tools
- The Colly framework for web scraping capabilities
