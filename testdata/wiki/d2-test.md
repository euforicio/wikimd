# D2 Diagram Playground

This page exercises the server-side D2 renderer. The first block explicitly opts into the ELK layout engine to ensure inline directives are respected:

```d2
vars: {
  d2-config: {
    layout-engine: elk
  }
}

app: Service
cache: Redis
store: Postgres
app -> cache -> store
```

The next block relies on the default (dagre) engine so we can visually compare the output:

```d2
client: Browser
proxy: Edge
api: API
client -> proxy -> api
```
