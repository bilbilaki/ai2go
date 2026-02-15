used +14000 ~token 

---
name: flutter-agent
description: Comprehensive Flutter development assistant for creating, debugging, and optimizing Flutter applications. Use when Codex needs to work with Flutter/Dart projects for: (1) Creating new Flutter apps, (2) Debugging Flutter issues, (3) Optimizing Flutter performance, (4) Implementing Flutter widgets and state management, (5) Working with pubspec.yaml dependencies, or any other Flutter development tasks.
metadata:
  short-description: Flutter development assistant
---
# Flutter Agent Skill
This skill provides comprehensive guidance for Flutter development, enabling Codex to assist with Flutter app creation, debugging, optimization, and maintenance.
## About Flutter Development
Flutter is Google's UI toolkit for building natively compiled applications for mobile, web, and desktop from a single codebase. This skill transforms Codex into a specialized Flutter development assistant equipped with procedural knowledge for Flutter/Dart projects.
### What This Skill Provides
1. **Flutter project setup** - Creating new Flutter apps with proper structure
2. **Widget implementation** - Building UI with Flutter's widget system
3. **State management** - Implementing various state management patterns (Provider, Riverpod, Bloc, etc.)
4. **Debugging assistance** - Identifying and fixing common Flutter issues
5. **Performance optimization** - Improving app performance and reducing jank
6. **Package management** - Working with pubspec.yaml and dependencies
7. **Platform integration** - Handling platform-specific code and native modules
## Core Principles for Flutter Development
### Dart Language Best Practices
**Default assumption: Codex understands Dart syntax.** Focus on Flutter-specific patterns and optimizations:
- Use `const` constructors for widgets when possible to improve performance
- Prefer `final` variables for immutable data
- Use sound null safety patterns
- Follow Dart style guide (effective-dart)
### Flutter Architecture Patterns
Match the architecture to the app's complexity:
**Simple apps**: Use basic StatefulWidget/StatelessWidget with Provider
**Medium complexity**: Use Riverpod or Bloc for state management
**Complex apps**: Consider Clean Architecture with feature-based organization
Think of Flutter development as building with LEGO blocks: widgets compose together, and state management determines how data flows through the structure.
### Anatomy of a Flutter Project
Every Flutter project follows a standard structure:
```
flutter-app/
├── lib/                    # Dart source code
│   ├── main.dart          # App entry point
│   ├── widgets/           # Reusable UI components
│   ├── screens/           # Full screen widgets
│   ├── models/            # Data models
│   ├── services/          # Business logic and API calls
│   └── utils/             # Helper functions
├── pubspec.yaml           # Dependencies and metadata
├── android/               # Android-specific code
├── ios/                   # iOS-specific code
├── web/                   # Web-specific code
└── test/                  # Test files
```
#### pubspec.yaml (required)
Every Flutter project has a pubspec.yaml that defines:
- **Dependencies**: Packages from pub.dev
- **Flutter configuration**: Assets, fonts, permissions
- **App metadata**: Name, description, version
#### lib/main.dart (required)
The entry point that sets up the MaterialApp/CupertinoApp and initial route.
#### Platform directories (optional)
- Platform-specific code goes in android/, ios/, web/ directories
- Use platform channels for native functionality
## Essential Flutter Concepts
### Widget Tree
Flutter UIs are built as a widget tree. Key widget types:
- **StatelessWidget**: Immutable widgets that depend only on their configuration
- **StatefulWidget**: Widgets that can change over time, with associated State objects
- **InheritedWidget**: Widgets that propagate information down the tree
### Build Context
The `BuildContext` provides widget location in the tree and access to inherited widgets.
### State Management
Choose based on app needs:
1. **setState**: Simple local state
2. **Provider**: Simple dependency injection and state management
3. **Riverpod**: Improved Provider with compile-time safety
4. **Bloc**: Business Logic Component pattern for complex state
5. **GetX**: Lightweight but opinionated framework
### Navigation
- **Navigator 1.0**: Imperative navigation with push/pop
- **Navigator 2.0**: Declarative navigation with Router API
- **go_router**: Popular package for simplified navigation
## Common Flutter Tasks
### Creating a New Flutter App
```bash
flutter create my_app
cd my_app
flutter run
```
### Adding Dependencies
Edit pubspec.yaml:
```yaml
dependencies:
  flutter:
    sdk: flutter
  provider: ^6.1.3
  http: ^1.1.0
```
Then run:
```bash
flutter pub get
```
### Building a Simple Widget
```dart
class MyHomePage extends StatelessWidget {
  const MyHomePage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Flutter App'),
      ),
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: <Widget>[
            const Text('Hello, Flutter!'),
            ElevatedButton(
              onPressed: () {
                // Handle button press
              },
              child: const Text('Press Me'),
            ),
          ],
        ),
      ),
    );
  }
}
```
### Debugging Common Issues
1. **Hot reload not working**: Ensure you're in debug mode, check for syntax errors
2. **Render overflow errors**: Use SingleChildScrollView or ListView for scrollable content
3. **State not updating**: Verify setState() is called or state management is properly set up
4. **Package version conflicts**: Run `flutter pub deps` to see dependency tree
## Performance Optimization
### Build Optimization
- Use `const` constructors for widgets that don't change
- Split large build methods into smaller widgets
- Use `ListView.builder` for long lists
- Avoid rebuilding entire trees unnecessarily
### Memory Management
- Dispose controllers and streams in `dispose()` method
- Use `AutomaticKeepAliveClientMixin` for preserving state
- Profile with DevTools memory tab
### Rendering Performance
- Use `RepaintBoundary` to isolate expensive paints
- Minimize opacity and clipping operations
- Use `Transform` instead of Positioned for animations
## Testing
### Unit Tests
```dart
test('Counter increments', () {
  final counter = Counter();
  counter.increment();
  expect(counter.value, 1);
});
```
### Widget Tests
```dart
testWidgets('MyWidget has a title', (WidgetTester tester) async {
  await tester.pumpWidget(const MyWidget());
  expect(find.text('Title'), findsOneWidget);
});
```
### Integration Tests
```dart
test('App starts and shows home page', () async {
  await tester.pumpAndSettle();
  expect(find.text('Home'), findsOneWidget);
});
```
## Platform-Specific Code
### Method Channels
```dart
// Dart side
const platform = MethodChannel('samples.flutter.dev/battery');
final int batteryLevel = await platform.invokeMethod('getBatteryLevel');
```
```kotlin
// Android side (Kotlin)
override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
  super.configureFlutterEngine(flutterEngine)
  MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "samples.flutter.dev/battery")
    .setMethodCallHandler { call, result ->
      if (call.method == "getBatteryLevel") {
        val batteryLevel = getBatteryLevel()
        result.success(batteryLevel)
      } else {
        result.notImplemented()
      }
    }
}
```
## Deployment
### Building for Release
```bash
# Android
flutter build apk --release
flutter build appbundle --release
# iOS
flutter build ios --release
# Web
flutter build web --release
```
### App Signing
- Android: Configure signing in android/app/build.gradle
- iOS: Use Xcode for code signing and provisioning
## Resources
### Essential Packages
- **http**: For API calls
- **provider/riverpod**: State management
- **shared_preferences**: Local storage
- **url_launcher**: Opening URLs
- **image_picker**: Selecting images
- **camera**: Camera access
- **geolocator**: Location services
- **firebase_core**: Firebase integration
### Development Tools
- **Flutter DevTools**: Debugging and profiling
- **VS Code/Android Studio**: IDEs with Flutter extensions
- **flutter doctor**: Diagnosing environment issues
### Learning Resources
- **Flutter Documentation**: docs.flutter.dev
- **Flutter Samples**: github.com/flutter/samples
- **Pub.dev**: Package repository
- **Flutter Community**: Medium publications and blogs
## When to Use This Skill
Use this Flutter Agent skill when:
1. User asks about Flutter/Dart development
2. Need to create, debug, or optimize Flutter apps
3. Working with Flutter widgets, state management, or architecture
4. Handling platform-specific Flutter code
5. Managing Flutter dependencies and packages
6. Building, testing, or deploying Flutter applications
This skill provides the specialized knowledge needed to transform Codex into an expert Flutter development assistant.
