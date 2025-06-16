# Email Harvester Frontend

This is the frontend application for the Email Harvester service, built with React, TypeScript, and Material-UI.

## Prerequisites

- Node.js 18 or later
- npm 8 or later

## Environment Variables

Create a `.env` file in the root directory with the following variables:

```env
REACT_APP_API_URL=http://localhost:8080
REACT_APP_ENV=development
```

## Available Scripts

In the project directory, you can run:

### `npm start`

Runs the app in development mode.\
Open [http://localhost:3000](http://localhost:3000) to view it in the browser.

### `npm test`

Launches the test runner in interactive watch mode.

### `npm run build`

Builds the app for production to the `build` folder.

### `npm run lint`

Runs ESLint to check for code style issues.

### `npm run format`

Runs Prettier to format the code.

## Project Structure

```
frontend/
├── public/              # Static files
├── src/
│   ├── api/            # API client and types
│   ├── components/     # Reusable components
│   ├── pages/         # Page components
│   ├── utils/         # Utility functions
│   ├── App.tsx        # Main app component
│   ├── index.tsx      # Entry point
│   └── theme.ts       # Material-UI theme
├── .eslintrc.json     # ESLint configuration
├── .prettierrc        # Prettier configuration
├── package.json       # Dependencies and scripts
└── tsconfig.json      # TypeScript configuration
```

## Development

1. Install dependencies:
   ```bash
   npm install
   ```

2. Start the development server:
   ```bash
   npm start
   ```

3. The application will be available at http://localhost:3000

## Building for Production

1. Build the application:
   ```bash
   npm run build
   ```

2. The production build will be available in the `build` directory.

## Docker

The application can be built and run using Docker:

```bash
docker build -t email-harvester-frontend .
docker run -p 3000:3000 email-harvester-frontend
```

Or using Docker Compose (from the root directory):

```bash
docker-compose up frontend
```

## Code Style

This project uses ESLint and Prettier for code style enforcement. The configuration files are:

- `.eslintrc.json` - ESLint rules and configuration
- `.prettierrc` - Prettier formatting rules

To check for code style issues:
```bash
npm run lint
```

To automatically fix code style issues:
```bash
npm run format
``` 