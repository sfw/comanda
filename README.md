# COMandA (Chain of Models and Actions)

COMandA is a command-line tool that enables the composition of Large Language Model (LLM) operations using a YAML-based Domain Specific Language (DSL). It simplifies the process of creating and managing chains of LLM activities that operate on files and information.

## Features

- 🔗 Chain multiple LLM operations together using simple YAML configuration
- 🤖 Support for multiple LLM providers (OpenAI, Anthropic, Ollama)
- 📄 File-based operations and transformations
- 🛠️ Extensible DSL for defining complex workflows
- ⚡ Efficient processing of LLM chains

## Installation

```bash
go install github.com/kris-hansen/comanda@latest
```

Or clone and build from source:

```bash
git clone https://github.com/kris-hansen/comanda.git
cd comanda
go build
```

## Configuration

Configure your providers and models using the interactive configuration command:

```bash
comanda configure
```

This will prompt you to:
1. Select a provider (OpenAI/Anthropic/Ollama)
2. Enter API key (for OpenAI/Anthropic)
3. Specify model name

You can view your current configuration using:

```bash
comanda configure --list
```

Example configuration output:
```yaml
providers:
  openai:
    api_key: sk-...
    models:
      - name: gpt-4
        type: external
  anthropic:
    api_key: sk-...
    models:
      - name: claude-2
        type: external
  ollama:
    models:
      - name: llama2
        type: local
```

## Usage

1. Create a YAML file defining your chain of operations:

```yaml
# example.yaml
steps:
  - name: "Summarize Content"
    model: "gpt-4"
    provider: "openai"
    input: 
      file: "input.txt"
    prompt: "Summarize the following content:"
    output:
      file: "summary.txt"

  - name: "Generate Analysis"
    model: "claude-2"
    provider: "anthropic"
    input:
      file: "summary.txt"
    prompt: "Analyze the key points in this summary:"
    output:
      file: "analysis.txt"
```

2. Run the chain:

```bash
comanda process -f your_chain.yaml
```

For example:

```bash
./comanda process example-dsl.yaml

Processing DSL file: example-dsl.yaml

Response from gpt-4o-mini:
Based on the company names provided, the following seem more like startups, often characterized by modern, innovative, and tech-oriented names:

1. Quantum Computing Labs
2. Blue Ocean Ventures
3. CloudNine Solutions
4. GreenField Robotics
5. Phoenix Digital Group
6. Sapphire Software Corp
7. Crystal Clear Analytics
8. Cascade Technologies
9. Velocity Ventures
10. Prism Technologies
11. Falcon Security Systems
12. Emerald Technologies
13. Unity Software Corp
14. Horizon Healthcare
15. Cyber Defense Labs
16. Infinity Software
17. Oasis Digital Group
18. Quantum Leap Systems
19. Solar Systems Corp
20. Vanguard Solutions
21. Zephyr Technologies
22. Aegis Solutions
23. Echo Technologies
24. Helix Technologies
25. Impact Software
26. Kinetic Solutions
27. Neural Networks Inc
28. Quest Software
29. Radiant Solutions
30. Yield Technologies

These names typically reflect a focus on technology, innovation, and modern solutions, which are common traits of startups.

Configuration:
- Model: [gpt-4o-mini]
- Action: [look through these company names and identify which ones seem like startups]
- Output: [STDOUT]
```


## Project Structure

```
comanda/
├── cmd/                    # Command line interface
├── utils/
│   ├── config/            # Configuration handling
│   ├── input/             # Input validation and processing
│   ├── models/            # LLM provider implementations
│   └── processor/         # DSL processing logic
├── go.mod
├── go.sum
└── main.go
```

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

## Acknowledgments

- OpenAI and Anthropic for their LLM APIs
- The Ollama project for local LLM support
- The Go community for excellent libraries and tools
