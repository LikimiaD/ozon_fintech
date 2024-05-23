# Ozon Fintech GraphQL API

## Features

* View list of posts
* View a specific post and its comments
* Users can disable comments for their posts
* Hierarchical comments with unlimited nesting
* Comment text limited to 2000 characters
* Pagination for comments
* Asynchronous delivery of new comments using GraphQL subscriptions

## Requirements

* Go 1.22+
* PostgreSQL (optional, for persistent storage)
* Redis
* Docker and Docker Compose (for containerized deployment)

## Getting Started

### Configuration

The application uses environment variables for configuration. Create a .env file in the root directory with the following content:

```dotenv
DB_NAME=ozon_fintech
DB_USER=postgres
DB_PASSWORD=ozon_fintech_password
DB_PORT=5432
DB_HOST=postgres
REDIS_ADDRESS=redis:6379
REDIS_PASSWORD=ozon_fintech_redis_password
REDIS_DB=0
HTTP_PORT=8080
```

### Running Locally

1. Install dependencies:

    ```shell
    make download
    ```

2. Build and run the application:

    ```shell
    make local_build
    ```
   
3. [Open link](http://localhost:8080/) in your browser to access the GraphQL playground.

### Running with Docker

1. Build and run the application:

    ```shell
    make docker
    ```

2. [Open link](http://localhost:8080/) in your browser to access the GraphQL playground.

## GraphQL Schema

```graphql
type Post {
    id: ID!
    title: String!
    content: String!
    author: String!
    commentsEnabled: Boolean!
    comments: [Comment!]!
    createdAt: String!
    updatedAt: String!
}

type Comment {
    id: ID!
    postId: ID!
    commentId: ID
    author: String!
    content: String!
    createdAt: String!
    updatedAt: String!
    replies: [Comment]
}

type Query {
    posts: [Post!]!
    post(id: ID!): Post
}

type Mutation {
    createPost(title: String!, content: String!, author: String!, commentsEnabled: Boolean!): Post
    updatePost(id: ID!, title: String, content: String, commentsEnabled: Boolean): Post
    deletePost(id: ID!): Boolean

    createComment(postId: ID!, commentId: ID, author: String!, content: String!): Comment
    updateComment(id: ID!, content: String!): Comment
    deleteComment(id: ID!): Boolean
}

type Subscription {
    commentAdded(postId: ID!): Comment!
}
```

## Endpoints

```text
/docs/   -> GraphQL documentation
/        -> GraphQL playground
/queries -> GraphQL queries
```
