# Architectural Decision Records

This file is a simple architecture decision log (ADL) of ADRs for the current project.

If an ADR conflicts with an organization-level ADR, the project-level ADR takes precedence.

<!-- 
## Title

### Status

What is the status, such as proposed, accepted, rejected, deprecated, superseded, etc.?

### Context

What is the issue that we're seeing that is motivating this decision or change?

### Decision

What is the change that we're proposing and/or doing?

### Consequences

What becomes easier or more difficult to do because of this change?
-->

## Task IDs should be unique across all topics

### Status

Accepted

### Context

Tasks in a Ratus deployment often serve the same purpose, but have different priorities or scheduling arrangements. For example, periodic tasks could be divided into two topics: `initial` and `subsequent`. After the initial execution of a task, it could be **transferred** to the `subsequent` topic, which has lower priority.

The topic mechanism also enables pipelined processing, allowing consumers at different stages to pass unique task descriptors to each other. For example, the `building` consumer in a CI/CD workflow could **transfer** a task descriptor to the `releasing` consumer without pushing to another queue.

### Decision

Task IDs across all topics should share the same namespace, and **a topic is just a named subset of the tasks**. When necessary, use different Ratus deployments to separate namespaces.

### Consequences

This decision significantly reduces duplicate information and redundant components in the system and saves storage space, allowing efficient and reliable transfer of tasks across topics. But on the other hand, additional checks are required to ensure the uniqueness of the IDs when inserting tasks.

## Support OCC via the nonce field

### Status

Accepted

### Context

On sharded collections, some MongoDB commands have strict requirements for query conditions:

> * When using [`findAndModify`](https://www.mongodb.com/docs/v4.4/reference/method/db.collection.findAndModify/#sharded-collections) against a sharded collection, the query must **contain an equality condition on shard key**.
> * When using [`updateOne`](https://www.mongodb.com/docs/v4.4/reference/method/db.collection.updateOne/#sharded-collections) on a sharded collection:
>    * If you don't specify `upsert: true`, you must **include an exact match on the `_id` field** *or* target a single shard (such as by including the shard key in the filter).
>    * If you specify `upsert: true`, the filter must **include the shard key**.
> * When using [`deleteOne`](https://www.mongodb.com/docs/v4.4/reference/method/db.collection.deleteOne/#sharded-collections) against a sharded collection, the query must **include the `_id` field** *or* the shard key.
> * When using [`replaceOne`](https://www.mongodb.com/docs/v4.4/reference/method/db.collection.replaceOne/#upsert-on-a-sharded-collection) that includes `upsert: true` on a sharded collection, the query must **include the full shard key**.

These requirements make it impossible to use atomic operations in some specific cases. To avoid unintended data changes between two or more consecutive operations, a strategy is required to track the state of the data.

In addition, since Ratus allows multiple consumers to run simultaneously, a strategy is required to invalidate duplicated commits.

### Decision

In order to implement [optimistic concurrency control](https://en.wikipedia.org/wiki/Optimistic_concurrency_control) (OCC), let each operation that may cause a change in the data to generate a random nonce string that is stored along with the data. Subsequent operations can **verify nonce strings to ensure data consistency**.

### Consequences

This decision solves two problems at once:
1. Improve compatibility with different sharding strategies.
2. Prevent unintended commits to tasks.

The added complexity is generally manageable.
