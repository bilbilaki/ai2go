class Ai2GoConfig {
  const Ai2GoConfig({
    required this.baseUrl,
    required this.apiKey,
    required this.currentModel,
    this.timeoutSeconds = 120,
    this.autoAccept = true,
  });

  final String baseUrl;
  final String apiKey;
  final String currentModel;
  final int timeoutSeconds;
  final bool autoAccept;
}
