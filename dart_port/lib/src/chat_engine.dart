import 'chat_history.dart';
import 'chat_models.dart';
import 'client.dart';
import 'tools.dart';

class ChatEngine {
  ChatEngine({
    required this.client,
    required this.toolExecutor,
    List<ToolDefinition>? tools,
  }) : tools = tools ?? Ai2GoTools.defaults();

  final Ai2GoClient client;
  final ToolExecutor toolExecutor;
  final List<ToolDefinition> tools;

  Future<ChatMessage> processTurn(ChatHistory history, String userInput) async {
    history.addUser(userInput);

    while (true) {
      final assistant = await client.runCompletion(
        history: history.messages,
        tools: tools,
      );
      history.addAssistant(assistant);

      if (assistant.toolCalls.isEmpty) {
        return assistant;
      }

      for (final call in assistant.toolCalls) {
        final output = await _handleToolCall(call);
        history.addToolResponse(call.id, output);
      }
    }
  }

  Future<String> _handleToolCall(ToolCall call) async {
    final args = call.function.decodeArguments();
    // Intentionally abstracted for plugin/app integration.
    return toolExecutor.execute(call.function.name, args);
  }
}
