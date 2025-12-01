Output walkthrough.gif

Set FontSize 24
Set Width 1200
Set Height 800
Set TypingSpeed 50ms

Type "# Install from chart (dev: no DB, 1 worker)"
Enter
Sleep 500ms
Type "cd demo"
Enter
Sleep 500ms
Type "composepack install charts/webapp --name app --auto-start"
Enter
Sleep 3s
Type "composepack ps app"
Enter
Sleep 3s
Type "cat charts/webapp/values.yaml"
Enter
Sleep 2s
Ctrl+L
Sleep 300ms

Type "# Update to production (with DB, 3 workers)"
Enter
Sleep 500ms
Type "cat values-prod.yaml"
Enter
Sleep 2s
Type "composepack install charts/webapp --name app -f values-prod.yaml --auto-start"
Enter
Sleep 10s
Type "composepack ps app"
Enter
Sleep 4s
Ctrl+L
Sleep 300ms

Type "# Package chart for distribution"
Enter
Sleep 500ms
Type "composepack package charts/webapp --destination dist/"
Enter
Sleep 1s
Type "ls -lh dist/"
Enter
Sleep 1s
Ctrl+L
Sleep 300ms

Type "# Update from packaged chart (staging: DB, 2 workers)"
Enter
Sleep 500ms
Type "cat values-staging.yaml"
Enter
Sleep 2s
Type "composepack install dist/webapp-1.0.0.cpack.tgz --name app -f values-staging.yaml --auto-start"
Enter
Sleep 5s
Type "composepack ps app"
Enter
Sleep 1s
Sleep 2s
Ctrl+L
Sleep 2s

