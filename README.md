# Load Test Tool

This project provides a load testing tool for evaluating the performance of web services. It uses [apib](https://github.com/apigee/apib) for generating load and is based off of Scott White's [Load Testing](https://medium.com/@scott_white/scalability-testing-74f30a875c1d) article.

## Features

- Perform load tests with varying concurrency levels.
- Predict optimal concurrency for a target latency.
- Generate plots for latency and requests per second (RPS).

## Requirements

- Go 1.22 or later

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/palmdalian/loadtest.git
    cd loadtest
    ```

2. Build and push the Docker image:
    ```sh
    docker buildx create --use
    docker buildx build --progress=plain \
        --platform=linux/amd64,linux/arm64 \
        -f ./Dockerfile \
        -t $IMAGE \
        -o type=image,push=true .
    ```

## Usage

- It's best practice to run the loadtester from within your infrastructure. From within the docker image:
    ```sh
    loadtester -url <URL> -duration <DURATION> -target <TARGET_LATENCY> -concurrency <CONCURRENCY_LEVELS> [-check] [-plot]
    ```

    - `-url`: The URL to test (required).
    - `-duration`: Duration of each test in seconds (default: 10).
    - `-target`: Target latency (ms) for prediction (default: 100).
    - `-concurrency`: Comma-separated list of concurrency levels (default: "1,2,10,50,100,200").
    - `-check`: Re-run apib to check prediction (optional).
    - `-plot`: Generate plots (optional).

- Example:
    ```sh
    loadtester -url http://example.com -duration 10 -target 100 -concurrency 1,2,10,50,100 -check -plot
    ```

## Running locally

1. Install [apib](https://github.com/apigee/apib/tree/master?tab=readme-ov-file#installation)

2. Run the load test:
    ```sh
    go run cmd/main.go -url <URL> -duration <DURATION> -target <TARGET_LATENCY> -concurrency <CONCURRENCY_LEVELS> [-check] [-plot]
    ```

    - `-url`: The URL to test (required).
    - `-duration`: Duration of each test in seconds (default: 10).
    - `-target`: Target latency (ms) for prediction (default: 100).
    - `-concurrency`: Comma-separated list of concurrency levels (default: "1,2,10,50,100,200").
    - `-check`: Re-run apib to check prediction (optional).
    - `-plot`: Generate plots (optional).

- Example:
    ```sh
    go run cmd/main.go -url http://example.com -duration 10 -target 100 -concurrency 1,2,10,50,100,200 -check -plot
    ```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## Contact

For any questions or feedback, please contact the project maintainer.
