# OpAMP Commander Documentation

This directory contains the official documentation for OpAMP Commander.

## Automatic Deployment

Documentation is automatically deployed to GitHub Pages when changes are pushed to the `main` branch. The deployment workflow builds the documentation using Hugo and publishes it to GitHub Pages.

View the live documentation at: `https://minuk-dev.github.io/opampcommander/`

## Local Development

To run the documentation site locally:

### Prerequisites

- [Hugo](https://gohugo.io/installation/) v0.170.1 or later
- [Dart Sass](https://gohugo.io/functions/css/sass/#dart-sass) with `sass` on your `PATH`
- Node.js 24 and npm for PostCSS processing

### Running Locally

```bash
# Start development server
hugo server -D

# Or using npm
npm install
npm run dev
```

The documentation site will be available at `http://localhost:1313`.

## Building

To create a production build:

```bash
npm install
hugo --minify
```

Built files will be generated in the `public/` directory.

## Writing Documentation

To add a new documentation page, create a markdown file in the `content/en/docs/` directory.

```bash
hugo new content/en/docs/your-section/your-page.md
```

## Theme

This documentation uses the [Docsy](https://www.docsy.dev/) theme.

## Contributing

We welcome contributions to improve the documentation! Please submit a Pull Request.
