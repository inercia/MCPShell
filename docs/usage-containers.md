# Running MCPShell in Containers

This document explains how to build specialized container images for running MCPShell as
an MCP server, and how to deploy them in container orchestration platforms like
Kubernetes.

## Building a Container Image with MCPShell

Container images provide a portable and reproducible way to package MCPShell along with
your tool configurations. This approach is particularly useful for:

- **Consistent deployments** across different environments
- **Version control** of both MCPShell and your tool configurations
- **Easy scaling** in container orchestration platforms
- **Isolation** from the host system

### Multi-Stage Dockerfile Example

Here's a simplified multi-stage Dockerfile that builds MCPShell and packages it with a
configuration file:

```dockerfile
# Stage 1: Build MCPShell
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Clone and build MCPShell
RUN git clone https://github.com/inercia/MCPShell.git . && \
    make build

# Stage 2: Runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache bash curl jq ca-certificates

# Copy the MCPShell binary from builder stage
COPY --from=builder /build/build/mcpshell /usr/local/bin/mcpshell

# Copy your configuration file
COPY tools.yaml /etc/mcpshell/tools.yaml

# Set default command
ENTRYPOINT ["mcpshell"]
CMD ["mcp", "--tools", "/etc/mcpshell/tools.yaml", "--http", "--port", "8080"]
```

### Directory Structure for Building

When building your image, organize your files like this:

```
my-mcp-server/
├── Dockerfile
├── tools.yaml          # Your MCPShell configuration
└── configs/            # Optional: additional config files
    └── descriptions.md
```

Build the image with:

```bash
docker build -t my-mcp-server:latest .
```

### Advanced Dockerfile Tips

#### Using a Specific Version

Pin to a specific MCPShell version for reproducibility:

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /build
RUN git clone --branch v0.1.5 --depth 1 https://github.com/inercia/MCPShell.git . && \
    make build
```

#### Multi-Configuration Support

Support multiple tool configurations:

```dockerfile
# Copy multiple configuration files
COPY configs/*.yaml /etc/mcpshell/

# Use environment variable to select config
ENV MCPSHELL_CONFIG=/etc/mcpshell/tools.yaml
CMD ["sh", "-c", "mcpshell mcp --tools ${MCPSHELL_CONFIG} --http --port 8080"]
```

#### Including Additional Tools

Install additional command-line tools your configuration might need:

```dockerfile
FROM alpine:latest
RUN apk add --no-cache \
    bash \
    curl \
    jq \
    kubectl \
    aws-cli \
    git \
    ca-certificates

COPY --from=builder /build/build/mcpshell /usr/local/bin/mcpshell
COPY tools.yaml /etc/mcpshell/tools.yaml
```

#### Security Hardening

Run as non-root user:

```dockerfile
FROM alpine:latest
RUN apk add --no-cache bash curl ca-certificates && \
    addgroup -g 1000 mcpshell && \
    adduser -D -u 1000 -G mcpshell mcpshell && \
    mkdir -p /etc/mcpshell && \
    chown -R mcpshell:mcpshell /etc/mcpshell

USER mcpshell
COPY --from=builder --chown=mcpshell:mcpshell /build/build/mcpshell /usr/local/bin/mcpshell
COPY --chown=mcpshell:mcpshell tools.yaml /etc/mcpshell/tools.yaml

ENTRYPOINT ["mcpshell"]
CMD ["mcp", "--tools", "/etc/mcpshell/tools.yaml", "--http", "--port", "8080"]
```

#### Health Checks

Add a health check endpoint for container orchestration:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1
```

**Note**: Ensure your MCPShell server exposes a health endpoint if using this feature.

## Running in Kubernetes

Kubernetes provides excellent orchestration capabilities for running MCPShell servers at
scale.

### Basic Deployment Example

Here's a simplified Kubernetes deployment that uses a ConfigMap for the tools
configuration:

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcpshell-config
  namespace: default
data:
  tools.yaml: |
    mcp:
      description: |
        Collection of diagnostic and monitoring tools
      run:
        shell: bash
      tools:
        - name: "system_info"
          description: "Get system information"
          run:
            command: "uname -a"
            timeout: "10s"
        
        - name: "disk_usage"
          description: "Check disk usage"
          params:
            path:
              type: string
              description: "Path to check"
              default: "/"
          constraints:
            - "path.startsWith('/')"
          run:
            command: "df -h {{ .path }}"
            timeout: "10s"

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcpshell-server
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mcpshell
  template:
    metadata:
      labels:
        app: mcpshell
    spec:
      containers:
        - name: mcpshell
          image: my-mcp-server:latest
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 8080
              name: http
              protocol: TCP
          volumeMounts:
            - name: config
              mountPath: /etc/mcpshell
              readOnly: true
          env:
            - name: LOG_LEVEL
              value: "info"
          resources:
            requests:
              memory: "64Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "500m"
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
      volumes:
        - name: config
          configMap:
            name: mcpshell-config

---
apiVersion: v1
kind: Service
metadata:
  name: mcpshell-service
  namespace: default
spec:
  selector:
    app: mcpshell
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
  type: ClusterIP
```

### Deployment Tips

#### Using Secrets for Sensitive Data

If your tools need access to credentials or API keys, use Kubernetes Secrets:

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: mcpshell-secrets
  namespace: default
type: Opaque
stringData:
  aws-credentials: |
    [default]
    aws_access_key_id = YOUR_ACCESS_KEY
    aws_secret_access_key = YOUR_SECRET_KEY

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcpshell-server
spec:
  template:
    spec:
      containers:
        - name: mcpshell
          image: my-mcp-server:latest
          volumeMounts:
            - name: config
              mountPath: /etc/mcpshell
              readOnly: true
            - name: secrets
              mountPath: /secrets
              readOnly: true
          env:
            - name: AWS_SHARED_CREDENTIALS_FILE
              value: "/secrets/aws-credentials"
      volumes:
        - name: config
          configMap:
            name: mcpshell-config
        - name: secrets
          secret:
            secretName: mcpshell-secrets
```

#### Namespace Isolation

Deploy different MCPShell configurations to different namespaces for isolation:

```bash
# Development environment
kubectl create namespace mcp-dev
kubectl apply -f mcpshell-deployment.yaml -n mcp-dev

# Production environment
kubectl create namespace mcp-prod
kubectl apply -f mcpshell-deployment.yaml -n mcp-prod
```

#### Ingress for External Access

Expose your MCPShell server externally with an Ingress:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: mcpshell-ingress
  namespace: default
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
    - host: mcp.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: mcpshell-service
                port:
                  number: 80
```

#### Horizontal Pod Autoscaling

Automatically scale based on resource usage:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: mcpshell-hpa
  namespace: default
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: mcpshell-server
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

### Monitoring and Logging

#### Centralized Logging

Configure MCPShell to output logs that can be collected by your logging infrastructure:

```yaml
spec:
  template:
    spec:
      containers:
        - name: mcpshell
          args:
            - "mcp"
            - "--tools"
            - "/etc/mcpshell/tools.yaml"
            - "--http"
            - "--port"
            - "8080"
            - "--log-level"
            - "info"
```

Logs will be available through `kubectl logs`:

```bash
kubectl logs -f deployment/mcpshell-server -n default
```

#### ServiceMonitor for Prometheus

If using Prometheus Operator, create a ServiceMonitor:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: mcpshell-metrics
  namespace: default
spec:
  selector:
    matchLabels:
      app: mcpshell
  endpoints:
    - port: http
      interval: 30s
      path: /metrics
```

## Best Practices

1. **Version your images**: Always tag your Docker images with specific versions, not
   just `latest`
2. **Resource limits**: Set appropriate CPU and memory limits to prevent resource
   exhaustion
3. **Health checks**: Implement proper health and readiness probes
4. **Security**: Run containers as non-root users and use read-only filesystems where
   possible
5. **Configuration management**: Use ConfigMaps for configurations and Secrets for
   sensitive data
6. **Logging**: Configure appropriate log levels and ensure logs are collected by your
   logging infrastructure
7. **Monitoring**: Expose metrics and set up appropriate alerts
8. **Updates**: Keep MCPShell and base images updated for security patches

## See Also

- [Configuration Guide](config.md): Learn about MCPShell configuration options
- [Runner Configuration](config-runners.md): Understand different execution runners
- [Security Considerations](security.md): Important security practices
- [Usage Guide](usage.md): Command-line usage and options
