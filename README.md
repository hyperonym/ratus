# Ratus

[![Go](https://github.com/hyperonym/ratus/actions/workflows/go.yml/badge.svg)](https://github.com/hyperonym/ratus/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/hyperonym/ratus/branch/master/graph/badge.svg?token=6HJKAQ9XR1)](https://codecov.io/gh/hyperonym/ratus)
[![Go Reference](https://pkg.go.dev/badge/github.com/hyperonym/ratus.svg)](https://pkg.go.dev/github.com/hyperonym/ratus)
[![Swagger Validator](https://img.shields.io/swagger/valid/3.0?specUrl=https%3A%2F%2Fraw.githubusercontent.com%2Fhyperonym%2Fratus%2Fmaster%2Fdocs%2Fswagger.json)](https://hyperonym.github.io/ratus/)
[![Go Report Card](https://goreportcard.com/badge/github.com/hyperonym/ratus)](https://goreportcard.com/report/github.com/hyperonym/ratus)
[![Status](https://img.shields.io/badge/status-alpha-yellow)](https://github.com/hyperonym/ratus)

Ratus is a RESTful asynchronous task queue service. It translated concepts of distributed task queues into a set of resources that conform to [REST principles](https://en.wikipedia.org/wiki/Representational_state_transfer) and provides an easy-to-use HTTP API.

The key features of Ratus are:

* Guaranteed at-least-once execution of tasks.
* Automatic recovery of timed out tasks.
* Simple language agnostic RESTful API.
* Time-based task scheduling.
* Naturally load balanced across consumers.
* Support dynamic topology changes.
* Scaling through replication and partitioning.
* Pluggable storage engine architecture.
* Prometheus integration for observability.

## Concepts

### Data Model

* **Task** references an idempotent unit of work that should be executed asynchronously.
* **Topic** refers to an ordered subset of tasks with the same topic name property.
* **Promise** represents a claim on the ownership of an active task.
* **Commit** contains a set of updates to be applied to a task.

### Workflow

* **Producer** client pushes **tasks** with their desired date-of-execution (scheduled times) to a **topic**.
* **Consumer** client makes a **promise** to execute a **task** polled from a **topic** and acknowledges with a **commit** upon completion.

### Topology

* Both **producer** and **consumer** clients can have multiple instances running simultaneously.
* **Consumer** instances can be added dynamically to increase throughput, and **tasks** will be naturally load balanced among consumers.
* **Consumer** instances can be removed (or crash) at any time without risking to lose the task being executing: a **task** that has not received a **commit** after the **promised** deadline will be picked up and executed again by other consumers.

### Task States

* **pending** (0): The task is ready to be executed or is waiting to be executed in the future.
* **active** (1): The task is being processed by a consumer. Active tasks that have timed out will be automatically reset to the `pending` state. Consumer code should handle failure and set the state to `pending` to retry later if necessary.
* **completed** (2): The task has completed its execution. If the storage engine implementation supports TTL, completed tasks will be automatically deleted after the retention period has expired.
* **archived** (3): The task is stored as an archive. Archived tasks will never be deleted due to expiration.

### Behavior

* **Task IDs across all topics share the same namespace** ([ADR](https://github.com/hyperonym/ratus/blob/master/docs/ARCHITECTURAL_DECISION_RECORDS.md#task-ids-should-be-unique-across-all-topics)). Topics are simply subsets generated based on the `topic` properties of the tasks, so topics do not need to be created explicitly.
* Ratus is a task scheduler when consumers can keep up with the task generation speed, or a priority queue when consumers cannot keep up with the task generation speed.
* Tasks will not be executed until the scheduled time arrives. After the scheduled time, excessive tasks will be executed in the order of the scheduled time.

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
