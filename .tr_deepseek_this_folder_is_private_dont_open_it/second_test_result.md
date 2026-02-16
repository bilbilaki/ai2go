used 11780 ~token 

---
name: flutter-agent
description: Specialized skill for Flutter development tasks including widget creation, state management, pubspec.yaml configuration, Dart programming, and Flutter project architecture. Use this skill when users ask about Flutter app development, UI implementation, package management, or debugging Flutter/Dart code.
metadata:
  short-description: Flutter development expertise
---
# Flutter Agent Skill
This skill provides specialized knowledge and workflows for Flutter development, transforming Codex into a Flutter expert capable of handling complex mobile and cross-platform app development tasks.
## About Flutter Development
Flutter is Google's UI toolkit for building natively compiled applications for mobile, web, and desktop from a single codebase using the Dart programming language. This skill equips Codex with deep knowledge of:
1. **Widget Architecture** - Stateless vs Stateful widgets, built-in widgets, custom widgets
2. **State Management** - Provider, Riverpod, Bloc, GetX, and other patterns
3. **Dart Programming** - Language features, async/await, streams, isolates
4. **Package Ecosystem** - Pub.dev packages, dependency management, versioning
5. **Project Structure** - Best practices for organizing Flutter projects
6. **Debugging & Testing** - Flutter DevTools, widget testing, integration testing
## Core Principles for Flutter Development
### Follow Flutter Best Practices
- **Widget Composition**: Prefer composition over inheritance, build small reusable widgets
- **State Management**: Choose appropriate state management based on app complexity
- **Performance**: Use const constructors, avoid unnecessary rebuilds, implement efficient lists
- **Null Safety**: Always use sound null safety, handle nullable types properly
### Project Structure Guidelines
A typical Flutter project should be organized as:
```
lib/
├── main.dart
├── models/
│   ├── user.dart
│   └── product.dart
├── services/
│   ├── api_service.dart
│   └── storage_service.dart
├── widgets/
│   ├── custom_button.dart
│   └── loading_indicator.dart
├── screens/
│   ├── home_screen.dart
│   └── detail_screen.dart
├── utils/
│   ├── constants.dart
│   └── helpers.dart
└── state/
    ├── providers/
    └── blocs/
```
### Pubspec.yaml Management
The pubspec.yaml file is critical for Flutter projects. Key sections include:
```yaml
name: my_flutter_app
description: A new Flutter project
publish_to: 'none'
version: 1.0.0+1
environment:
  sdk: '>=3.0.0 <4.0.0'
  flutter: '>=3.0.0'
dependencies:
  flutter:
    sdk: flutter
  cupertino_icons: ^1.0.6
  provider: ^6.1.1
  http: ^1.1.0
dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^3.0.0
flutter:
  uses-material-design: true
  assets:
    - assets/images/
  fonts:
    - family: Roboto
      fonts:
        - asset: assets/fonts/Roboto-Regular.ttf
```
## Workflow Patterns
### Creating a New Flutter Widget
When creating a new widget, follow this pattern:
```dart
import 'package:flutter/material.dart';
class CustomWidget extends StatelessWidget {
  final String title;
  final VoidCallback onPressed;
  const CustomWidget({
    super.key,
    required this.title,
    required this.onPressed,
  });
  @override
  Widget build(BuildContext context) {
    return ElevatedButton(
      onPressed: onPressed,
      child: Text(title),
    );
  }
}
```
### State Management with Provider
For simple to medium complexity apps, Provider is recommended:
```dart
// Provider class
class CounterProvider extends ChangeNotifier {
  int _count = 0;
  int get count => _count;
  void increment() {
    _count++;
    notifyListeners();
  }
}
// Usage in widget
class CounterScreen extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: Consumer<CounterProvider>(
          builder: (context, counter, child) {
            return Text('Count: ${counter.count}');
          },
        ),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () {
          context.read<CounterProvider>().increment();
        },
        child: Icon(Icons.add),
      ),
    );
  }
}
```
## Common Tasks & Solutions
### 1. Adding Dependencies
Always check pub.dev for the latest versions and update pubspec.yaml accordingly. Run `flutter pub get` after changes.
### 2. Handling API Calls
Use the http package with proper error handling:
```dart
Future<User> fetchUser() async {
  final response = await http.get(
    Uri.parse('https://api.example.com/user'),
  );
  if (response.statusCode == 200) {
    return User.fromJson(jsonDecode(response.body));
  } else {
    throw Exception('Failed to load user');
  }
}
```
### 3. Navigation
Use Navigator 2.0 for complex navigation or Navigator 1.0 for simple apps:
```dart
Navigator.push(
  context,
  MaterialPageRoute(
    builder: (context) => DetailScreen(item: item),
  ),
);
```
### 4. Theming
Define app theme in MaterialApp:
```dart
MaterialApp(
  theme: ThemeData(
    primarySwatch: Colors.blue,
    visualDensity: VisualDensity.adaptivePlatformDensity,
  ),
  home: HomeScreen(),
);
```
## Debugging & Optimization
### Common Issues
1. **RenderFlex overflow**: Use SingleChildScrollView or ListView
2. **State not updating**: Ensure notifyListeners() is called
3. **Slow performance**: Use const widgets, implement ListView.builder
4. **Hot reload not working**: Check for syntax errors or restart app
### Performance Tips
- Use `const` constructors for widgets that don't change
- Implement `ListView.builder` for long lists
- Use `AutomaticKeepAliveClientMixin` for state preservation
- Profile with Flutter DevTools Performance tab
## Testing
### Widget Tests
```dart
testWidgets('Counter increments', (WidgetTester tester) async {
  await tester.pumpWidget(MyApp());
  expect(find.text('0'), findsOneWidget);
  await tester.tap(find.byIcon(Icons.add));
  await tester.pump();
  expect(find.text('1'), findsOneWidget);
});
```
## When to Use This Skill
Use this Flutter Agent skill when:
1. User asks about Flutter app development
2. Creating or modifying Flutter widgets
3. Setting up state management
4. Configuring pubspec.yaml dependencies
5. Debugging Flutter/Dart code
6. Implementing Flutter architecture patterns
7. Optimizing Flutter app performance
8. Adding packages from pub.dev
## Resources
For detailed references, check:
- `references/flutter_widgets.md` - Complete widget catalog
- `references/state_management.md` - State management patterns comparison
- `references/dart_language.md` - Dart language features and syntax
- `references/pubspec_reference.md` - Complete pubspec.yaml specification
## Quick Reference Commands
```bash
# Create new Flutter project
flutter create my_app
# Run app
flutter run
# Get packages
flutter pub get
# Run tests
flutter test
# Build APK
flutter build apk
# Build iOS
flutter build ios
# Clean project
flutter clean
```
This skill transforms Codex into a Flutter development expert capable of handling any Flutter-related task with professional-grade solutions.
