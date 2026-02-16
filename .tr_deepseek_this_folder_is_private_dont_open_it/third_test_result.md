used 18118 ~token

---
name: flutter-agent
description: Guide for AI agents working with Flutter development. This skill should be used when users need assistance with Flutter/Dart projects, including widget creation, state management, pubspec.yaml configuration, and Flutter-specific tooling.
metadata:
  short-description: Flutter development assistant
---
# Flutter Agent Skill
This skill provides specialized knowledge and workflows for Flutter development, enabling AI agents to effectively assist with Flutter/Dart projects.
## About Flutter Development
Flutter is Google's UI toolkit for building natively compiled applications for mobile, web, and desktop from a single codebase. This skill equips AI agents with Flutter-specific knowledge including widget architecture, state management patterns, Dart language features, and Flutter tooling.
### What This Skill Provides
1. **Widget Architecture** - Understanding of Stateless vs Stateful widgets, built-in widgets, and custom widget creation
2. **State Management** - Patterns like Provider, Riverpod, Bloc, GetX, and when to use each
3. **Dart Language** - Dart-specific syntax, null safety, async/await patterns, and language features
4. **Tool Integration** - Working with pubspec.yaml, Flutter CLI commands, package management, and debugging
## Core Principles for Flutter Assistance
### Understand Flutter's Declarative UI
Flutter uses a declarative UI paradigm where the UI is rebuilt with new state. Help users understand:
- Widgets are immutable descriptions of UI
- State objects hold mutable data that triggers rebuilds
- The widget tree vs element tree vs render tree
### Prioritize Performance Best Practices
Flutter apps need to maintain 60fps (or 120fps for promoted devices). Guide users to:
- Use const constructors where possible
- Implement efficient build methods
- Use appropriate state management for their scale
- Profile with DevTools for performance bottlenecks
### Match Solution Complexity to Project Scale
**Simple apps**: Provider or Riverpod for state management
**Medium apps**: Bloc or GetX with proper architecture
**Large apps**: Clean Architecture with dependency injection
## Anatomy of a Flutter Project
```
flutter_project/
├── lib/
│   ├── main.dart          # App entry point
│   ├── app/              # App configuration
│   ├── features/         # Feature-based modules
│   ├── shared/           # Shared widgets & utilities
│   └── core/             # Core business logic
├── pubspec.yaml         # Dependencies & metadata
├── android/             # Android-specific code
├── ios/                 # iOS-specific code
├── web/                 # Web-specific code
└── test/                # Test files
```
### Key Files to Understand
- **pubspec.yaml**: The project configuration file defining dependencies, assets, and metadata
- **main.dart**: Application entry point with MaterialApp/CupertinoApp
- **Widget classes**: Understanding StatelessWidget vs StatefulWidget
## Common Flutter Tasks
### 1. Creating New Widgets
**Stateless Widget Template:**
```dart
class MyWidget extends StatelessWidget {
  const MyWidget({super.key, required this.title});
  
  final String title;
  
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      child: Text(title),
    );
  }
}
```
**Stateful Widget Template:**
```dart
class CounterWidget extends StatefulWidget {
  const CounterWidget({super.key});
  
  @override
  State<CounterWidget> createState() => _CounterWidgetState();
}
class _CounterWidgetState extends State<CounterWidget> {
  int _counter = 0;
  
  void _incrementCounter() {
    setState(() {
      _counter++;
    });
  }
  
  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text('Count: $_counter'),
        ElevatedButton(
          onPressed: _incrementCounter,
          child: const Text('Increment'),
        ),
      ],
    );
  }
}
```
### 2. Working with pubspec.yaml
**Essential sections:**
```yaml
name: my_flutter_app
description: A sample Flutter application
publish_to: 'none'
version: 1.0.0+1
environment:
  sdk: '>=3.0.0 <4.0.0'
  flutter: '>=3.0.0'
dependencies:
  flutter:
    sdk: flutter
  cupertino_icons: ^1.0.2
  provider: ^6.0.0  # State management
  http: ^1.0.0      # HTTP requests
dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^3.0.0
flutter:
  uses-material-design: true
  assets:
    - assets/images/
```
### 3. State Management Selection Guide
| Pattern | Best For | Complexity | Learning Curve |
|---------|----------|------------|----------------|
| **Provider** | Small to medium apps | Low | Easy |
| **Riverpod** | All app sizes | Medium | Moderate |
| **Bloc** | Medium to large apps | High | Steep |
| **GetX** | Rapid development | Low | Easy |
| **MobX** | Reactive programming | Medium | Moderate |
## Flutter CLI Commands Reference
```bash
# Create new project
flutter create my_app
# Run app
flutter run
# Build for platforms
flutter build apk
flutter build ios
flutter build web
# Add packages
flutter pub add package_name
# Generate files
flutter pub run build_runner build
# Analyze code
flutter analyze
# Format code
dart format .
```
## Debugging & Performance
### Common Issues & Solutions
1. **Render overflow errors**: Use SingleChildScrollView, ListView, or constrain containers
2. **Performance jank**: Profile with DevTools, use const widgets, implement didUpdateWidget
3. **State not updating**: Ensure setState() is called, check widget tree rebuilds
4. **Package conflicts**: Run `flutter pub upgrade` or resolve version constraints
### DevTools Features
- **Widget Inspector**: Examine widget tree
- **Performance View**: Identify rendering bottlenecks
- **Memory View**: Detect memory leaks
- **Network View**: Monitor HTTP traffic
## Testing in Flutter
### Test Types
1. **Unit tests**: Test individual functions/methods
2. **Widget tests**: Test UI components in isolation
3. **Integration tests**: Test complete app flows
### Example Widget Test:
```dart
void main() {
  testWidgets('Counter increments', (WidgetTester tester) async {
    await tester.pumpWidget(const MyApp());
    
    expect(find.text('0'), findsOneWidget);
    expect(find.text('1'), findsNothing);
    
    await tester.tap(find.byIcon(Icons.add));
    await tester.pump();
    
    expect(find.text('0'), findsNothing);
    expect(find.text('1'), findsOneWidget);
  });
}
```
## When to Use This Skill
This skill should be triggered when users ask about:
- Flutter widget creation or customization
- State management patterns and implementation
- Dart language features and syntax
- pubspec.yaml configuration and dependencies
- Flutter project structure and architecture
- Performance optimization in Flutter
- Testing Flutter applications
- Flutter CLI commands and tooling
- Platform-specific implementations (Android/iOS/Web)
## Quick Reference Patterns
### Navigation
```dart
Navigator.push(
  context,
  MaterialPageRoute(builder: (context) => const NextScreen()),
);
```
### HTTP Requests
```dart
final response = await http.get(Uri.parse('https://api.example.com/data'));
if (response.statusCode == 200) {
  return jsonDecode(response.body);
} else {
  throw Exception('Failed to load data');
}
```
### Form Handling
```dart
final _formKey = GlobalKey<FormState>();
final _controller = TextEditingController();
@override
Widget build(BuildContext context) {
  return Form(
    key: _formKey,
    child: Column(
      children: [
        TextFormField(
          controller: _controller,
          validator: (value) {
            if (value == null || value.isEmpty) {
              return 'Please enter some text';
            }
            return null;
          },
        ),
        ElevatedButton(
          onPressed: () {
            if (_formKey.currentState!.validate()) {
              // Process data
            }
          },
          child: const Text('Submit'),
        ),
      ],
    ),
  );
}
```
## Resources
- [Flutter Documentation](https://flutter.dev/docs)
- [Dart Language Tour](https://dart.dev/guides/language/language-tour)
- [Flutter Widget Catalog](https://flutter.dev/docs/development/ui/widgets)
- [Flutter API Reference](https://api.flutter.dev/)
- [Pub.dev Package Repository](https://pub.dev)
