# Contributing to CogniShell

First off, thank you for taking the time to contribute! We welcome contributions of all forms, including bug fixes, new features, documentation enhancements, and testing improvements.

This guide outlines our development workflow and coding standards to help you get started.

---

## 1. Development Workflow

### Step 1: Open or Claim an Issue
Before making any changes, please search active issues or open a new one. This ensures we don't duplicate efforts and allows for design discussions before code is written.

### Step 2: Create a Feature Branch
Create a branch from the `main` branch. Use a descriptive naming convention:
* `feature/your-feature-name` (for new features)
* `bugfix/issue-number-description` (for bug fixes)
* `docs/short-description` (for documentation changes)
* `refactor/short-description` (for code cleanups)

```bash
git checkout -b feature/dynamic-secret-injection
```

### Step 3: Implement Your Changes
* Ensure your changes are focused and solve one specific issue.
* Adhere to the code style conventions defined below.
* Write unit tests for any new logic introduced.

### Step 4: Run Local Verification
Before submitting a pull request, ensure everything compiles, tests pass, and coding style matches:

* **Go (Backend)**:
  ```bash
  cd backend
  go fmt ./...
  go test -v ./...
  ```
* **Next.js (Frontend UI)**:
  ```bash
  cd frontend
  npm run lint
  ```

### Step 5: Open a Pull Request (PR)
1. Push your branch to GitHub.
2. Open a Pull Request targeting the `main` branch.
3. Fill out the PR template, referencing the related issue number (e.g., `Closes #42`).
4. Wait for a code review. Be ready to iterate on feedback.

---

## 2. Coding Conventions

### Go Backend Guidelines
* **Formatting**: Always format your Go code using `gofmt` before committing.
* **Error Handling**: Do not ignore errors. Always return them or log them appropriately. Wrap errors with context where necessary (`fmt.Errorf("retrieving secrets: %w", err)`).
* **Concurrency**: Ensure goroutines are managed using Contexts (`context.Context`) for graceful cancellation and timeout handling.
* **Logging**: Use standard logging or structural logging helpers. Avoid raw `println()` in production code.

### Next.js Frontend Guidelines
* **TypeScript**: Use strict typing. Avoid the use of `any` types wherever possible.
* **Components**: Keep UI components modular, clean, and responsive. Use CSS or styled themes consistently.
* **State Management**: Manage API query states (loading, success, error) cleanly to ensure a smooth user experience.

---

## 3. Secrets & Security Policy

* **Never Hardcode Secrets**: Under no circumstances should passwords, GitHub tokens, Vault AppRole IDs, or webhook URLs be committed to the repository.
* **Use Mock Credentials for Testing**: Make use of `.env.example` configurations with dummy placeholders for local validation.
