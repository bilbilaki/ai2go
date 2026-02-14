import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import 'chat_models.dart';
import 'config.dart';

class Ai2GoClient {
  Ai2GoClient(this.config, {http.Client? httpClient})
      : _httpClient = httpClient ?? http.Client();

  final Ai2GoConfig config;
  final http.Client _httpClient;

  Future<ChatMessage> runCompletion({
    required List<ChatMessage> history,
    required List<ToolDefinition> tools,
  }) async {
    final uri = Uri.parse('${config.baseUrl}/v1/chat/completions');
    final req = http.Request('POST', uri)
      ..headers['Authorization'] = 'Bearer ${config.apiKey}'
      ..headers['Content-Type'] = 'application/json'
      ..body = jsonEncode({
        'model': config.currentModel,
        'stream': true,
        'messages': history.map((e) => e.toJson()).toList(),
        'tools': tools.map((e) => e.toJson()).toList(),
      });

    final streamed = await _httpClient.send(req).timeout(
          Duration(seconds: config.timeoutSeconds),
        );

    if (streamed.statusCode != 200) {
      final body = await streamed.stream.bytesToString();
      throw Exception('API error ${streamed.statusCode}: $body');
    }

    return _fromSse(streamed.stream.transform(utf8.decoder));
  }

  Future<ChatMessage> _fromSse(Stream<String> stream) async {
    final toolCalls = <String, ToolCall>{};
    final order = <String>[];
    var content = '';

    await for (final chunk in stream) {
      final lines = const LineSplitter().convert(chunk);
      for (final line in lines) {
        if (!line.startsWith('data: ')) continue;
        final data = line.substring(6).trim();
        if (data == '[DONE]') {
          return ChatMessage(
            role: 'assistant',
            content: content,
            toolCalls: order.map((k) => toolCalls[k]!).toList(),
          );
        }

        Map<String, dynamic> parsed;
        try {
          parsed = jsonDecode(data) as Map<String, dynamic>;
        } catch (_) {
          continue;
        }

        final choices = (parsed['choices'] as List?) ?? const [];
        for (var i = 0; i < choices.length; i++) {
          final delta = (choices[i] as Map<String, dynamic>)['delta'] as Map<String, dynamic>? ?? const {};
          content += (delta['content'] ?? '') as String;

          final tcList = (delta['tool_calls'] as List?) ?? const [];
          for (var idx = 0; idx < tcList.length; idx++) {
            final raw = tcList[idx] as Map<String, dynamic>;
            final key = (raw['id'] as String?)?.isNotEmpty == true ? raw['id'] as String : 'idx:$idx';
            final old = toolCalls[key];
            final fn = (raw['function'] as Map<String, dynamic>?) ?? const {};
            toolCalls[key] = ToolCall(
              id: (raw['id'] ?? old?.id ?? '') as String,
              type: (raw['type'] ?? old?.type ?? 'function') as String,
              function: FunctionCall(
                name: '${old?.function.name ?? ''}${(fn['name'] ?? '') as String}',
                arguments: '${old?.function.arguments ?? ''}${(fn['arguments'] ?? '') as String}',
              ),
            );
            if (!order.contains(key)) order.add(key);
          }
        }
      }
    }

    return ChatMessage(
      role: 'assistant',
      content: content,
      toolCalls: order.map((k) => toolCalls[k]!).toList(),
    );
  }
}
