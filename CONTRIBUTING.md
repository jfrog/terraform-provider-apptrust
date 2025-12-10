# Contributing to Terraform Provider for JFrog AppTrust

Thank you for your interest in contributing to the Terraform Provider for JFrog AppTrust!

## How to Contribute

### Reporting Issues

If you find a bug or have a feature request, please open an issue on GitHub with:
- Clear description of the issue
- Steps to reproduce (for bugs)
- Expected vs actual behavior
- Terraform and provider versions
- Relevant logs or error messages

### Submitting Changes

1. **Fork the repository**
2. **Create a feature branch** from `main`
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Follow Go coding standards
   - Add tests for new functionality
   - Update documentation as needed
   - Ensure all tests pass

4. **Commit your changes**
   ```bash
   git commit -m "Add: description of your change"
   ```
   Use clear, descriptive commit messages.

5. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Open a Pull Request**
   - Provide a clear description of changes
   - Reference any related issues
   - Ensure CI checks pass

## Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/jfrog/terraform-provider-apptrust.git
   cd terraform-provider-apptrust
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the provider:
   ```bash
   make build
   ```

4. Run tests:
   ```bash
   make test
   ```

## Code Standards

- Follow Go best practices and conventions
- Use `gofmt` for code formatting
- Add comments for exported functions and types
- Write unit tests for new code
- Ensure acceptance tests pass (if applicable)

## Testing

- Unit tests: `make test`
- Acceptance tests: `make acceptance` (requires Artifactory instance)
- See `TESTING.md` for detailed testing information

## Documentation

- Update documentation for any API changes
- Keep README.md up to date

## Questions?

Feel free to open an issue for questions or reach out to the maintainers.

Thank you for contributing!

