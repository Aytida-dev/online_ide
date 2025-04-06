const doc = {
  "main.go": {
    "description": "This file implements a production-grade server for managing Docker containers to execute live code in various programming languages. It leverages the Fiber framework for HTTP and WebSocket handling, providing a scalable and efficient solution for interactive coding sessions.",
    "key_components": {
      "DockerManager": "A struct responsible for managing Docker containers, including creation, scaling, and resource monitoring.",
      "LangOptions": "Defines configuration options for supported programming languages, such as Docker image, execution commands, and resource limits.",
      "WebSocket Integration": "Enables real-time communication for live coding sessions, allowing users to send and receive code execution results interactively."
    },
    "features": [
      "Dynamic container creation and reuse based on language and user demand.",
      "Resource monitoring and scaling to optimize container performance.",
      "Support for multiple programming languages with customizable configurations.",
      "Graceful server shutdown to ensure proper cleanup of resources."
    ],
    "usage": {
      "Start Server": "Run the application to start the server on port 3000.",
      "WebSocket Endpoint": "Connect to the `/ws` endpoint with a query parameter `language` to specify the programming language (e.g., `?language=js`).",
      "Interactive Coding": "Send code prefixed with 'CODE:' via WebSocket to execute it in a container."
    },
    "dependencies": {
      "Fiber": "Web framework for building fast and scalable web applications.",
      "Docker SDK": "Go client for interacting with the Docker API.",
      "WebSocket": "Library for handling WebSocket connections"
    }
  }
}
