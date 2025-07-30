# Mock Data Guide

This document explains how to use the built-in mock data feature of the Oracle DR Monitoring Dashboard.

## What is Mock Mode?

Mock Mode is a special feature that allows you to run the application's frontend without a live connection to an Oracle database. When activated, the backend serves pre-defined, simulated data that mimics a real-world monitoring environment.

This data includes various database states, roles, and connection statuses, providing a realistic and dynamic experience for development and testing.

## Why Use Mock Mode?

Mock Mode is incredibly useful for:

-   **Frontend Development**: Frontend developers can build and test UI components without needing access to a live database or worrying about the backend setup.
-   **UI/UX Testing**: Test how the dashboard handles different scenarios, such as database failures, high replication lag, or role transitions.
-   **Live Demos**: Showcase the application's features to stakeholders without exposing a production environment.
-   **CI/CD Pipelines**: Run automated UI tests in an environment without database dependencies.

## How to Enable Mock Mode

Enabling Mock Mode is a two-step process:

### 1. Build with the `mock` Tag

You must compile or run the application using the `mock` build tag. This tag ensures that the mock data API handlers are included in the binary.

**To run with mock support:**
```bash
go run -tags mock .
```

**To build a binary with mock support:**
```bash
go build -tags mock -o oracle-dr-dashboard_with_mock
```

If you build the application without the `mock` tag, the mock data API endpoint (`/api/mock-data`) will not be available.

### 2. Activate via URL Parameter

Once the application is running with mock support, you can activate the mock data feed by adding the `?mock=true` query parameter to the URL.

**Example URL:**
`http://localhost:8080/?mock=true`

## Mock Mode and Multi-Language Support

Mock Mode works seamlessly with the multi-language feature. The mock data API provides translated UI titles for all supported languages (English, Chinese, Japanese).

To see translated mock data, simply combine the `mock` and `lang` parameters in the URL.

**Examples:**
-   **Chinese**: `http://localhost:8080/?mock=true&lang=zh`
-   **Japanese**: `http://localhost:8080/?mock=true&lang=ja`
-   **English**: `http://localhost:8080/?mock=true&lang=en`
