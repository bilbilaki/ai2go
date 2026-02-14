# ai2go_dart

Dart/Flutter-friendly port of the core `ai2go` AI loop:
- chat history + system prompt
- streaming completion parsing
- tool-call loop handling
- tool schemas compatible with function-calling chat APIs

## Important note
Tool execution is intentionally **placeholder-only** in this port. You get tool schema + dispatch flow, but real tool behavior should be implemented in your host app/plugin.

## Flutter/plugin integration idea
1. Keep `ChatEngine` in your shared Dart package.
2. Implement `ToolExecutor` in your app/plugin layer.
3. Route tool calls to platform channels / backend APIs / local sandbox as needed.

## Quick usage
```dart
final engine = ChatEngine(
  client: Ai2GoClient(config),
  toolExecutor: MyToolExecutor(), // implement yourself
);
final history = ChatHistory(currentModel: config.currentModel);
final reply = await engine.processTurn(history, userInput);
```
