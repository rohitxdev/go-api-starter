# GitHub Copilot Instructions

- Be extremely concise. Sacrifice grammar for concision.
- Prefer clear, idiomatic, and maintainable code.
- Match existing project style and naming.
- Handle errors and edge cases cleanly with wrapped errors (`fmt.Errorf("context: %w", err)`).
- Never hard-code secrets or credentials.
- Prioritize readability over brevity.
- Suggest simple, readable, table-driven tests for new code.
