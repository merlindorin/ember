# 0013. Every RPC uses Connect

- Status: Accepted
- Date: 2026-10-27
- Area: tools
- Owner: user

## Context

ember's automation ran a plain grpc-go server, while the user's stack is Connect.

## Decision

- Automation and `SceneService` are served with connect-go, with Connect's reflection. No grpc-go anywhere.
- Code is generated with buf and `protoc-gen-connect-go`.

## Alternatives

- grpc-go: the default for Go, but no browser support and a second RPC stack.

## Consequences

Studio, browsers and grpcurl reach a running game the same way.
