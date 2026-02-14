import 'chat_models.dart';

class ChatHistory {
  ChatHistory({required this.currentModel}) {
    _messages.add(
      ChatMessage(
        role: 'system',
        content: 'You are an advanced terminal assistant. Model: $currentModel. '
            'Always plan briefly, use tools for actions, and avoid huge outputs.',
      ),
    );
  }

  final String currentModel;
  final List<ChatMessage> _messages = [];

  List<ChatMessage> get messages => List.unmodifiable(_messages);

  void addUser(String content) => _messages.add(ChatMessage(role: 'user', content: content));

  void addAssistant(ChatMessage message) => _messages.add(message);

  void addToolResponse(String toolCallId, String content) {
    _messages.add(
      ChatMessage(
        role: 'tool',
        toolCallId: toolCallId,
        content: content.isEmpty ? 'Tool executed successfully (no output).' : content,
      ),
    );
  }
}
