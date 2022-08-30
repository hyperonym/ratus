# Ratus

[![Go](https://github.com/hyperonym/ratus/actions/workflows/go.yml/badge.svg)](https://github.com/hyperonym/ratus/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/hyperonym/ratus/branch/master/graph/badge.svg?token=6HJKAQ9XR1)](https://codecov.io/gh/hyperonym/ratus)
[![Go Reference](https://pkg.go.dev/badge/github.com/hyperonym/ratus.svg)](https://pkg.go.dev/github.com/hyperonym/ratus)
[![Swagger Validator](https://img.shields.io/swagger/valid/3.0?specUrl=https%3A%2F%2Fraw.githubusercontent.com%2Fhyperonym%2Fratus%2Fmaster%2Fdocs%2Fswagger.json)](https://hyperonym.github.io/ratus/)
[![Go Report Card](https://goreportcard.com/badge/github.com/hyperonym/ratus)](https://goreportcard.com/report/github.com/hyperonym/ratus)
[![Status](https://img.shields.io/badge/status-alpha-yellow)](https://github.com/hyperonym/ratus)

Ratus is a RESTful asynchronous task queue service. It translated concepts of distributed task queues into a set of resources that conform to [REST principles](https://en.wikipedia.org/wiki/Representational_state_transfer) and provides an easy-to-use HTTP API.

## Observability

### Metrics and Labels

Ratus exposes [Prometheus](https://prometheus.io) metrics via HTTP, on the `/metrics` endpoint.

| Name | Type | Labels |
| --- | --- | --- |
| **ratus_request_duration_seconds** | histogram | `topic`, `endpoint`, `status_code` |
| **ratus_chore_duration_seconds** | histogram | - |
| **ratus_task_schedule_delay_seconds** | gauge | `topic`, `producer`, `consumer` |
| **ratus_task_execution_duration_seconds** | gauge | `topic`, `producer`, `consumer` |
| **ratus_task_produced_count_total** | counter | `topic`, `producer` |
| **ratus_task_consumed_count_total** | counter | `topic`, `producer`, `consumer` |
| **ratus_task_committed_count_total** | counter | `topic`, `producer`, `consumer` |

### Liveness and Readiness

Ratus supports [liveness and readiness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) via HTTP GET requests:

* The `/livez` endpoint returns a status code of **200** if the instance is running.
* The `/readyz` endpoint returns a status code of **200** if the instance is ready to accept traffic.

## Caveats

* 🚨 **Topic names and task IDs must not contain plus signs ('+') due to [gin-gonic/gin#2633](https://github.com/gin-gonic/gin/issues/2633).**
* It is not recommended to use Ratus as the main storage of tasks. Instead, consider storing the complete task record in a database, and **use a minimal descriptor as the payload for Ratus.**
* The `completed` state only indicates that the task has been executed, it does not mean the task was successful.
* Ratus is a simple and reliable alternative to task queues like [Celery](https://docs.celeryq.dev/). Consider to use [RabbitMQ](https://www.rabbitmq.com/) or [Kafka](https://kafka.apache.org/) if you need high-throughput message passing without task management.

## Frequently Asked Questions

For more details, see [Architectural Decision Records](https://github.com/hyperonym/ratus/blob/master/docs/ARCHITECTURAL_DECISION_RECORDS.md).

### Why HTTP API?

> Asynchronous task queues are typically used for long background tasks, so the overhead of the HTTP API is not significant compared to the time spent by the tasks themselves. On the other hand, the HTTP-based RESTful API can be easily accessed by all languages without using dedicated client libraries.

### How to poll from multiple topics?

> If the number of topics is limited and you don't care about the priority between them, you can choose to create multiple threads/goroutines to listen to them simultaneously. Alternatively, you can create a ***topic of topics*** to get the topic names in turn and then get the next task from the corresponding topic.
## Contributing

This project is open-source. If you have any ideas or questions, please feel free to reach out by creating an issue!

Contributions are greatly appreciated, please refer to [CONTRIBUTING.md](https://github.com/hyperonym/ratus/blob/master/CONTRIBUTING.md) for more information.

## License

Ratus is available under the [Mozilla Public License Version 2.0](https://github.com/hyperonym/ratus/blob/master/LICENSE).

---

© 2022 [Hyperonym](https://hyperonym.org)
