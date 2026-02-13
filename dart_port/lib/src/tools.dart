import 'chat_models.dart';

class Ai2GoTools {
  static List<ToolDefinition> defaults() => [
        runCommand,
        readFile,
        patchFile,
        searchFiles,
        listTree,
      ];

  static const runCommand = ToolDefinition(
    type: 'function',
    function: ToolFunction(
      name: 'run_command',
      description: 'Executes a shell command and returns output.',
      parameters: {
        'type': 'object',
        'properties': {
          'command': {'type': 'string'}
        },
        'required': ['command']
      },
    ),
  );

  static const readFile = ToolDefinition(
    type: 'function',
    function: ToolFunction(
      name: 'read_file',
      description: 'Reads a file and returns content with line numbers.',
      parameters: {
        'type': 'object',
        'properties': {
          'path': {'type': 'string'},
          'line_range': {'type': 'string'}
        },
        'required': ['path']
      },
    ),
  );

  static const patchFile = ToolDefinition(
    type: 'function',
    function: ToolFunction(
      name: 'patch_file',
      description: 'Applies line-based patch instructions to a file.',
      parameters: {
        'type': 'object',
        'properties': {
          'path': {'type': 'string'},
          'patch': {'type': 'string'}
        },
        'required': ['path', 'patch']
      },
    ),
  );

  static const searchFiles = ToolDefinition(
    type: 'function',
    function: ToolFunction(
      name: 'search_files',
      description: 'Advanced search by extension/path/content.',
      parameters: {
        'type': 'object',
        'properties': {
          'dir': {'type': 'string'},
          'ext': {'type': 'string'},
          'inc_path': {'type': 'string'},
          'exc_path': {'type': 'string'},
          'content': {'type': 'string'},
          'content_exclude': {'type': 'boolean'}
        },
        'required': ['dir']
      },
    ),
  );

  static const listTree = ToolDefinition(
    type: 'function',
    function: ToolFunction(
      name: 'list_tree',
      description: 'Generates a visual directory tree.',
      parameters: {
        'type': 'object',
        'properties': {
          'dir': {'type': 'string'}
        },
        'required': ['dir']
      },
    ),
  );
}

abstract interface class ToolExecutor {
  Future<String> execute(String name, Map<String, dynamic> arguments);
}

class PlaceholderToolExecutor implements ToolExecutor {
  const PlaceholderToolExecutor();

  @override
  Future<String> execute(String name, Map<String, dynamic> arguments) async {
    return '[placeholder] Tool "$name" called with args: $arguments';
  }
}
