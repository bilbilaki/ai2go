import 'dart:io';

import 'package:ai2go_dart/ai2go_dart.dart';

Future<void> main() async {
  final apiKey = Platform.environment['OPENAI_API_KEY'] ?? '';
  if (apiKey.isEmpty) {
    stderr.writeln('Set OPENAI_API_KEY first.');
    exitCode = 1;
    return;
  }

  final config = Ai2GoConfig(
    baseUrl: 'https://api.openai.com',
    apiKey: apiKey,
    currentModel: 'gpt-4o-mini',
  );

  final engine = ChatEngine(
    client: Ai2GoClient(config),
    toolExecutor: const PlaceholderToolExecutor(),
  );

  final history = ChatHistory(currentModel: config.currentModel);

  stdout.writeln('ai2go_dart demo (placeholder tools). Type "exit" to quit.');
  while (true) {
    stdout.write('> ');
    final input = stdin.readLineSync()?.trim() ?? '';
    if (input == 'exit' || input == 'quit') break;
    if (input.isEmpty) continue;

    final message = await engine.processTurn(history, input);
    stdout.writeln('assistant: ${message.content ?? ''}');
  }
}
