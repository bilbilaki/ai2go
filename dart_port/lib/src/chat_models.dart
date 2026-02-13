import 'dart:convert';

class ToolDefinition {
  const ToolDefinition({
    required this.type,
    required this.function,
  });

  final String type;
  final ToolFunction function;

  Map<String, dynamic> toJson() => {
        'type': type,
        'function': function.toJson(),
      };
}

class ToolFunction {
  const ToolFunction({
    required this.name,
    required this.description,
    required this.parameters,
  });

  final String name;
  final String description;
  final Map<String, dynamic> parameters;

  Map<String, dynamic> toJson() => {
        'name': name,
        'description': description,
        'parameters': parameters,
      };
}

class ChatMessage {
  const ChatMessage({
    required this.role,
    this.content,
    this.toolCalls = const [],
    this.toolCallId,
  });

  final String role;
  final String? content;
  final List<ToolCall> toolCalls;
  final String? toolCallId;

  Map<String, dynamic> toJson() {
    final map = <String, dynamic>{'role': role};
    if (content != null) map['content'] = content;
    if (toolCallId != null) map['tool_call_id'] = toolCallId;
    if (toolCalls.isNotEmpty) {
      map['tool_calls'] = toolCalls.map((e) => e.toJson()).toList();
    }
    return map;
  }

  static ChatMessage fromJson(Map<String, dynamic> json) => ChatMessage(
        role: json['role'] as String,
        content: json['content'] as String?,
        toolCallId: json['tool_call_id'] as String?,
        toolCalls: ((json['tool_calls'] as List?) ?? const [])
            .map((e) => ToolCall.fromJson(e as Map<String, dynamic>))
            .toList(),
      );
}

class ToolCall {
  const ToolCall({
    required this.id,
    required this.type,
    required this.function,
  });

  final String id;
  final String type;
  final FunctionCall function;

  Map<String, dynamic> toJson() => {
        'id': id,
        'type': type,
        'function': function.toJson(),
      };

  static ToolCall fromJson(Map<String, dynamic> json) => ToolCall(
        id: (json['id'] ?? '') as String,
        type: (json['type'] ?? 'function') as String,
        function: FunctionCall.fromJson(json['function'] as Map<String, dynamic>),
      );
}

class FunctionCall {
  const FunctionCall({
    required this.name,
    required this.arguments,
  });

  final String name;
  final String arguments;

  Map<String, dynamic> toJson() => {
        'name': name,
        'arguments': arguments,
      };

  static FunctionCall fromJson(Map<String, dynamic> json) => FunctionCall(
        name: (json['name'] ?? '') as String,
        arguments: (json['arguments'] ?? '') as String,
      );

  Map<String, dynamic> decodeArguments() {
    if (arguments.trim().isEmpty) return const {};
    return jsonDecode(arguments) as Map<String, dynamic>;
  }
}
